package sqlsplit

import (
	"fmt"
	"strings"
	"unicode"
)

type state int

const (
	stateStmt state = iota
	stateLineComment
	stateBlockComment
	stateString
	stateDollarString
	stateIdent
)

// Stmt represents a single SQL statement with its position in the original file.
type Stmt struct {
	Content string
	Start   Location
	End     Location
}

// Location tracks position, line number, and column in the input.
type Location struct {
	Pos  int
	Line int
	Col  int
}

func (l Location) String() string {
	return fmt.Sprintf("%d:%d", l.Line, l.Col)
}

type scanner struct {
	dollarTag  string
	buf        strings.Builder
	runes      []rune
	stmts      []Stmt
	currentLoc Location

	// startLoc is current statement start location
	startLoc Location

	// stateLoc is current state start location (e.g., can be a string within statement).
	stateLoc     Location
	state        state
	commentDepth int
}

func newScanner(sql string) *scanner {
	//nolint:exhaustruct
	return &scanner{
		runes:      []rune(sql),
		currentLoc: Location{Pos: 0, Line: 1, Col: 1},
	}
}

func (s *scanner) peek(offset int) rune {
	if s.currentLoc.Pos+offset < len(s.runes) {
		return s.runes[s.currentLoc.Pos+offset]
	}

	return 0
}

func (s *scanner) advance() {
	if s.runes[s.currentLoc.Pos] == '\n' {
		s.currentLoc.Line++
		s.currentLoc.Col = 1
	} else {
		s.currentLoc.Col++
	}

	s.currentLoc.Pos++
}

func (s *scanner) consume(n int) {
	for range n {
		s.buf.WriteRune(s.peek(0))
		s.advance()
	}
}

func (s *scanner) emitStmt() {
	if stmt := newStmt(s.buf.String(), s.startLoc, s.currentLoc); stmt != nil {
		s.stmts = append(s.stmts, *stmt)
	}

	s.buf.Reset()
}

func (s *scanner) scan() error {
	for s.currentLoc.Pos < len(s.runes) {
		switch s.state {
		case stateStmt:
			s.scanStmt()
		case stateLineComment:
			s.scanLineComment()
		case stateBlockComment:
			s.scanBlockComment()
		case stateString:
			s.scanString()
		case stateDollarString:
			s.scanDollarString()
		case stateIdent:
			s.scanIdent()
		}
	}

	if err := s.reportUnclosed(); err != nil {
		return err
	}

	s.emitStmt()

	return nil
}

func (s *scanner) reportUnclosed() error {
	switch s.state {
	case stateBlockComment:
		return fmt.Errorf("conduit: unclosed block comment starting at %s", s.stateLoc.String())
	case stateString:
		return fmt.Errorf("conduit: unclosed string starting at %s", s.stateLoc.String())
	case stateDollarString:
		if s.dollarTag == "" {
			return fmt.Errorf("conduit: unclosed dollar-quoted string starting at %s", s.stateLoc.String())
		}

		return fmt.Errorf("conduit: unclosed dollar-quoted string $%s$ starting at %s",
			s.dollarTag, s.stateLoc.String())
	case stateIdent:
		return fmt.Errorf("conduit: unclosed quoted identifier starting at %s", s.stateLoc.String())

	case stateStmt:
	case stateLineComment:
		// noop, all good
	}

	return nil
}

func (s *scanner) scanStmt() {
	r := s.peek(0)

	// Skip leading whitespace between statements
	if s.buf.Len() == 0 && unicode.IsSpace(r) {
		s.advance()
		return
	}

	// Track where actual statement content begins
	if s.buf.Len() == 0 {
		s.startLoc = s.currentLoc
	}

	switch {
	case r == '-' && s.peek(1) == '-':
		s.state = stateLineComment
		s.consume(2) // --

	case r == '/' && s.peek(1) == '*':
		s.stateLoc = s.currentLoc
		s.state = stateBlockComment
		s.commentDepth = 1
		s.consume(2) // /*

	case r == '$':
		if tag, endPos, ok := parseDollarQuoteTag(s.runes, s.currentLoc.Pos); ok {
			s.stateLoc = s.currentLoc
			s.state = stateDollarString
			s.dollarTag = tag
			s.consume(endPos - s.currentLoc.Pos)
		} else {
			s.consume(1)
		}

	case r == '\'':
		s.stateLoc = s.currentLoc
		s.state = stateString
		s.consume(1)

	case r == '"':
		s.stateLoc = s.currentLoc
		s.state = stateIdent
		s.consume(1)

	case r == ';':
		s.consume(1)
		s.emitStmt()

	default:
		s.consume(1)
	}
}

func (s *scanner) scanLineComment() {
	if s.peek(0) == '\n' {
		s.state = stateStmt
	}

	s.consume(1)
}

func (s *scanner) scanBlockComment() {
	switch {
	case s.peek(0) == '/' && s.peek(1) == '*':
		s.commentDepth++
		s.consume(2) // /*

	case s.peek(0) == '*' && s.peek(1) == '/':
		s.commentDepth--
		s.consume(2) // */

		if s.commentDepth == 0 {
			s.state = stateStmt
		}

	default:
		s.consume(1)
	}
}

func (s *scanner) scanString() {
	r := s.peek(0)

	switch {
	case r == '\\' && s.currentLoc.Pos+1 < len(s.runes):
		s.consume(2) // escape sequence

	case r == '\'' && s.peek(1) == '\'':
		s.consume(2) // ''

	case r == '\'':
		s.state = stateStmt
		s.consume(1)

	default:
		s.consume(1)
	}
}

func (s *scanner) scanDollarString() {
	if s.peek(0) == '$' {
		if closeTag, closeEnd, ok := parseDollarQuoteTag(s.runes, s.currentLoc.Pos); ok &&
			closeTag == s.dollarTag {
			s.state = stateStmt
			s.consume(closeEnd - s.currentLoc.Pos)

			return
		}
	}

	s.consume(1)
}

func (s *scanner) scanIdent() {
	r := s.peek(0)

	switch {
	case r == '"' && s.peek(1) == '"':
		s.consume(2) // ""

	case r == '"':
		s.state = stateStmt
		s.consume(1)

	default:
		s.consume(1)
	}
}

// Split splits a SQL file into individual statements.
func Split(sql string) ([]Stmt, error) {
	s := newScanner(sql)
	if err := s.scan(); err != nil {
		return nil, err
	}

	return s.stmts, nil
}

func newStmt(content string, start, end Location) *Stmt {
	content = strings.TrimSpace(content)
	if content == "" || content == ";" {
		return nil
	}

	return &Stmt{
		Content: content,
		Start:   start,
		End:     end,
	}
}

// parseDollarQuoteTag attempts to parse a dollar-quote tag at the given position.
// Returns (tag, endPos, ok) where tag is the identifier between $...$,
// endPos is the position after the closing $, and ok indicates if parsing succeeded.
// For $$, returns ("", pos+2, true). For $tag$, returns ("tag", pos+5, true).
func parseDollarQuoteTag(runes []rune, pos int) (string, int, bool) {
	if pos >= len(runes) || runes[pos] != '$' {
		return "", pos, false
	}

	start := pos
	pos++
	tagStart := pos

	// Empty tag: $$
	if pos < len(runes) && runes[pos] == '$' {
		return "", pos + 1, true
	}

	if pos >= len(runes) {
		return "", start, false
	}

	r := runes[pos]
	if !unicode.IsLetter(r) && r != '_' {
		return "", start, false
	}

	pos++

	for pos < len(runes) {
		r = runes[pos]
		if r == '$' {
			return string(runes[tagStart:pos]), pos + 1, true
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return "", start, false
		}

		pos++
	}

	return "", start, false
}
