//nolint:dupl
package sqlsplit

import (
	"testing"
	"unicode/utf8"
)

func TestPick(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		advances int
		wantRune rune
	}{
		{
			name:     "empty string",
			input:    "",
			advances: 0,
			wantRune: utf8.RuneError,
		},
		{
			name:     "single character no advance",
			input:    "a",
			advances: 0,
			wantRune: 'a',
		},
		{
			name:     "single character with advance",
			input:    "a",
			advances: 1,
			wantRune: utf8.RuneError,
		},
		{
			name:     "multiple characters no advance",
			input:    "abc",
			advances: 0,
			wantRune: 'a',
		},
		{
			name:     "multiple characters with one advance",
			input:    "abc",
			advances: 1,
			wantRune: 'b',
		},
		{
			name:     "multiple characters at end",
			input:    "abc",
			advances: 3,
			wantRune: utf8.RuneError,
		},
		{
			name:     "unicode character",
			input:    "こんにちは",
			advances: 0,
			wantRune: 'こ',
		},
		{
			name:     "special characters",
			input:    "!@#$",
			advances: 2,
			wantRune: '#',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))

			// Advance the lexer as specified
			for range tt.advances {
				l.advance()
			}

			gotRune := l.peek()
			if gotRune != tt.wantRune {
				t.Errorf("pick() rune = %v, want %v", string(gotRune), string(tt.wantRune))
			}
		})
	}
}

func TestAdvance(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		advances   int
		wantOffset int
		wantLine   int
		wantCol    int
	}{
		{
			name:       "empty string",
			input:      "",
			advances:   1,
			wantOffset: 0,
			wantLine:   1,
			wantCol:    1,
		},
		{
			name:       "single character",
			input:      "a",
			advances:   1,
			wantOffset: 1,
			wantLine:   1,
			wantCol:    2,
		},
		{
			name:       "multiple characters",
			input:      "abc",
			advances:   2,
			wantOffset: 2,
			wantLine:   1,
			wantCol:    3,
		},
		{
			name:       "newline \\n",
			input:      "a\nb",
			advances:   2,
			wantOffset: 2,
			wantLine:   2,
			wantCol:    1,
		},
		{
			name:       "newline \\r\\n",
			input:      "a\r\nb",
			advances:   3,
			wantOffset: 4,
			wantLine:   2,
			wantCol:    2,
		},
		{
			name:       "multiple newlines",
			input:      "a\nb\nc",
			advances:   4,
			wantOffset: 4,
			wantLine:   3,
			wantCol:    1,
		},
		{
			name:       "advance past end",
			input:      "ab",
			advances:   3,
			wantOffset: 2,
			wantLine:   1,
			wantCol:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))

			for range tt.advances {
				l.advance()
			}

			if l.offset != tt.wantOffset {
				t.Errorf("offset = %v, want %v", l.offset, tt.wantOffset)
			}

			if l.line != tt.wantLine {
				t.Errorf("line = %v, want %v", l.line, tt.wantLine)
			}

			if l.col != tt.wantCol {
				t.Errorf("col = %v, want %v", l.col, tt.wantCol)
			}
		})
	}
}

func TestNext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		advances int
		wantRune rune
	}{
		{
			name:     "empty string",
			input:    "",
			advances: 0,
			wantRune: utf8.RuneError,
		},
		{
			name:     "single character",
			input:    "a",
			advances: 0,
			wantRune: utf8.RuneError,
		},
		{
			name:     "two characters no advance",
			input:    "ab",
			advances: 0,
			wantRune: 'b',
		},
		{
			name:     "two characters with advance",
			input:    "ab",
			advances: 1,
			wantRune: utf8.RuneError,
		},
		{
			name:     "three characters middle",
			input:    "abc",
			advances: 1,
			wantRune: 'c',
		},
		{
			name:     "unicode characters",
			input:    "こんにちは",
			advances: 0,
			wantRune: 'ん',
		},
		{
			name:     "special characters",
			input:    "!@#",
			advances: 1,
			wantRune: '#',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))

			for range tt.advances {
				l.advance()
			}

			gotRune := l.next()
			if gotRune != tt.wantRune {
				t.Errorf("next() rune = %v, want %v", string(gotRune), string(tt.wantRune))
			}
		})
	}
}

func TestLexComment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStmt  Stmt
		wantCount int
	}{
		{
			name:  "simple comment",
			input: "--test comment\n",
			wantStmt: Stmt{
				kind:    Comment,
				content: "test comment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 15},
			},
			wantCount: 1,
		},
		{
			name:  "comment without newline",
			input: "--test comment",
			wantStmt: Stmt{
				kind:    Comment,
				content: "test comment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 15},
			},
			wantCount: 1,
		},
		{
			name:  "comment with carriage return",
			input: "--test comment\r\n",
			wantStmt: Stmt{
				kind:    Comment,
				content: "test comment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 15},
			},
			wantCount: 1,
		},
		{
			name:  "empty comment",
			input: "--\n",
			wantStmt: Stmt{
				kind:    Comment,
				content: "",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 3},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Lex()

			if len(l.stmts) != tt.wantCount {
				t.Errorf("got %d statements, want %d", len(l.stmts), tt.wantCount)
				return
			}

			got := l.stmts[0]
			if got.kind != tt.wantStmt.kind {
				t.Errorf("Kind = %v, want %v", got.kind, tt.wantStmt.kind)
			}

			if got.content != tt.wantStmt.content {
				t.Errorf("Content = %q, want %q", got.Content(), tt.wantStmt.Content())
			}

			if got.start != tt.wantStmt.start {
				t.Errorf("Start = %v, want %v", got.start, tt.wantStmt.start)
			}

			if got.end != tt.wantStmt.end {
				t.Errorf("End = %v, want %v", got.end, tt.wantStmt.end)
			}
		})
	}
}

func TestLexCommentMultiline(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStmt  Stmt
		wantCount int
	}{
		{
			name:  "simple multiline comment",
			input: "/*test comment*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "test comment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 17},
			},
			wantCount: 1,
		},
		{
			name:  "multiline comment with actual newlines",
			input: "/*test\ncomment*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "test\ncomment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 2, col: 10},
			},
			wantCount: 1,
		},
		{
			name:  "multiline comment with carriage returns",
			input: "/*test\r\ncomment*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "test\r\ncomment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 2, col: 10},
			},
			wantCount: 1,
		},
		{
			name:  "empty multiline comment",
			input: "/**/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 5},
			},
			wantCount: 1,
		},
		{
			name:  "multiline comment with multiple lines",
			input: "/*line 1\nline 2\nline 3*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "line 1\nline 2\nline 3",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 3, col: 9},
			},
			wantCount: 1,
		},
		{
			name:  "multiline comment with special characters",
			input: "/*SELECT * FROM table; -- nested comment*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "SELECT * FROM table; -- nested comment",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 43},
			},
			wantCount: 1,
		},
		{
			name:  "multiline comment with nested comments",
			input: "/*SELECT * FROM table; /*nested comment*/*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "SELECT * FROM table; /*nested comment*/",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 44},
			},
			wantCount: 1,
		},
		{
			name:  "utf8 comment",
			input: "/*SELECT * FROM таблица*/",
			wantStmt: Stmt{
				kind:    CommentMultiline,
				content: "SELECT * FROM таблица",
				start:   Pos{line: 1, col: 1},
				end:     Pos{line: 1, col: 26},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			l.Lex()

			if len(l.stmts) != tt.wantCount {
				t.Errorf("got %d statements, want %d", len(l.stmts), tt.wantCount)
				return
			}

			got := l.stmts[0]
			if got.kind != tt.wantStmt.kind {
				t.Errorf("Kind = %v, want %v", got.kind, tt.wantStmt.kind)
			}

			if got.content != tt.wantStmt.content {
				t.Errorf("Content = %q, want %q", got.Content(), tt.wantStmt.Content())
			}

			if got.start != tt.wantStmt.start {
				t.Errorf("Start = %v, want %v", got.start, tt.wantStmt.start)
			}

			if got.end != tt.wantStmt.end {
				t.Errorf("End = %v, want %v", got.end, tt.wantStmt.end)
			}
		})
	}
}
