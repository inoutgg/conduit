package sqlsplit

import (
	"unicode"
	"unicode/utf8"
)

type StmtKind int

const (
	Comment StmtKind = iota
	CommentMultiline
	Query
)

type Pos struct{ line, col int }

type Stmt struct {
	content string
	start   Pos
	end     Pos
	kind    StmtKind
}

func (s *Stmt) Kind() StmtKind  { return s.kind }
func (s *Stmt) Content() string { return s.content }
func (s *Stmt) Highlight(_ error) string {
	return ""
}

type Lexer struct {
	src   []byte
	stmts []*Stmt

	offset int

	line int
	col  int
}

func NewLexer(src []byte) *Lexer {
	return &Lexer{
		src:    src,
		offset: 0,
		line:   1,
		col:    1,
		stmts:  make([]*Stmt, 0),
	}
}

func (l *Lexer) peek() rune {
	if l.offset >= len(l.src) {
		return utf8.RuneError
	}

	r, _ := utf8.DecodeRune(l.src[l.offset:])

	return r
}

func (l *Lexer) advance() int {
	if l.offset >= len(l.src) {
		return 0
	}

	r, size := utf8.DecodeRune(l.src[l.offset:])
	if r == '\r' || r == '\n' {
		if l.next() == '\n' {
			l.offset++
		}

		l.line++
		l.col = 1
	} else {
		l.col++
	}

	l.offset += size

	return size
}

func (l *Lexer) next() rune {
	_, size := utf8.DecodeRune(l.src[l.offset:])
	r, _ := utf8.DecodeRune(l.src[l.offset+size:])

	return r
}

func (l *Lexer) skipWhitespace() bool {
	skipped := false

	for ch := l.peek(); unicode.IsSpace(ch); ch = l.peek() {
		l.advance()

		skipped = true
	}

	return skipped
}

func (l *Lexer) pos() Pos { return Pos{l.line, l.col} }

func (l *Lexer) add(s *Stmt) (int, Pos) {
	l.stmts = append(l.stmts, s)
	return l.offset, l.pos()
}

func (l *Lexer) lexComment() (int, Pos) {
	//nolint:exhaustruct
	stmt := Stmt{kind: Comment, start: l.pos()}
	start := l.offset + 2 // skip --

	for ch := l.peek(); ch != utf8.RuneError && ch != '\n' && ch != '\r'; ch = l.peek() {
		l.advance()
	}

	stmt.end = l.pos()
	stmt.content = string(l.src[start:l.offset])

	return l.add(&stmt)
}

func (l *Lexer) lexCommentMultiline() (int, Pos) {
	//nolint:exhaustruct
	stmt := Stmt{kind: CommentMultiline, start: l.pos()}
	start := l.offset + 2 // skip /*
	lvl := 0

	for ch := l.peek(); ch != utf8.RuneError; ch = l.peek() {
		if ch == '*' && l.next() == '/' {
			lvl--
			if lvl == 0 {
				break
			}

			// skipping here only * as the later call to advance will skip /
			l.advance() // skip *
		} else if ch == '/' && l.next() == '*' {
			lvl++
		}

		l.advance()
	}

	size := 0
	size += l.advance() // skip *
	size += l.advance() // skip /

	stmt.end = l.pos()
	stmt.content = string(l.src[start : l.offset-size])

	return l.add(&stmt)
}

func (l *Lexer) lexQuery(start int, startPos Pos) (int, Pos) {
	stmt := Stmt{
		kind:    Query,
		start:   startPos,
		end:     l.pos(),
		content: string(l.src[start:l.offset]),
	}

	return l.add(&stmt)
}

func (l *Lexer) Lex() []*Stmt {
	offset := l.offset
	pos := l.pos()

	for ch := l.peek(); ch != utf8.RuneError; ch = l.peek() {
		if l.skipWhitespace() {
			offset = l.offset
			pos = l.pos()
		}

		switch ch {
		case '-':
			if l.next() == '-' {
				offset, pos = l.lexComment()
			}
		case '/':
			if l.next() == '*' {
				offset, pos = l.lexCommentMultiline()
			}

		case ';':
			offset, pos = l.lexQuery(offset, pos)
		}
	}

	return l.stmts
}
