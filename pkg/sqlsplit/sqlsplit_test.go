package sqlsplit

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple statements",
			input: "SELECT 1; SELECT 2;",
		},
		{
			name:  "single-line comments",
			input: "SELECT 1; -- comment\nSELECT 2;",
		},
		{
			name: "multi-line comments",
			input: `SELECT 1; /* comment
spanning lines */ SELECT 2;`,
		},
		{
			name:  "nested multi-line comments",
			input: "SELECT 1; /* outer /* inner */ still outer */ SELECT 2;",
		},
		{
			name:  "single-quoted strings",
			input: "SELECT 'test'; SELECT 'it''s';",
		},
		{
			name:  "semicolon in string",
			input: "SELECT 'test; value'; SELECT 2;",
		},
		{
			name:  "dollar-quoted strings",
			input: "SELECT $$test$$; SELECT $$it's$$;",
		},
		{
			name:  "dollar-quoted with tag",
			input: "SELECT $tag$test$tag$; SELECT $foo$it's$foo$;",
		},
		{
			name:  "dollar-quoted with semicolon",
			input: "SELECT $$test; value$$; SELECT 2;",
		},
		{
			name:  "dollar-quoted with nested dollars",
			input: "SELECT $outer$test $$ value$outer$; SELECT 2;",
		},
		{
			name:  "quoted identifiers",
			input: `SELECT "table"; SELECT "col;umn";`,
		},
		{
			name:  "escaped quote in identifier",
			input: `SELECT "test""name"; SELECT 2;`,
		},
		{
			name: "CREATE FUNCTION with dollar quoting",
			input: `CREATE FUNCTION test() RETURNS integer AS $$
BEGIN
    RETURN 1;
END;
$$ LANGUAGE plpgsql;
SELECT 2;`,
		},
		{
			name: "complex nested quoting",
			input: `CREATE FUNCTION test() RETURNS text AS $func$
BEGIN
    RETURN $inner$It's a "test"; value$inner$;
END;
$func$ LANGUAGE plpgsql;`,
		},
		{
			name:  "escape string syntax",
			input: `SELECT E'test\nvalue'; SELECT 'normal';`,
		},
		{
			name:  "empty statements",
			input: ";;; SELECT 1; ;;",
		},
		{
			name:  "trailing statement without semicolon",
			input: "SELECT 1; SELECT 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmts, err := Split(tt.input)
			require.NoError(t, err)
			snaps.MatchSnapshot(t, stmts)
		})
	}
}

func TestLocationTracking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []struct{ start, end Location }
	}{
		{
			name:  "single line multiple statements",
			input: "SELECT 1; SELECT 2;",
			expected: []struct{ start, end Location }{
				{Location{Pos: 0, Line: 1, Col: 1}, Location{Pos: 9, Line: 1, Col: 10}},
				{Location{Pos: 10, Line: 1, Col: 11}, Location{Pos: 19, Line: 1, Col: 20}},
			},
		},
		{
			name:  "statements on separate lines",
			input: "SELECT 1;\nSELECT 2;",
			expected: []struct{ start, end Location }{
				{Location{Pos: 0, Line: 1, Col: 1}, Location{Pos: 9, Line: 1, Col: 10}},
				{Location{Pos: 10, Line: 2, Col: 1}, Location{Pos: 19, Line: 2, Col: 10}},
			},
		},
		{
			name:  "indented statements",
			input: "  SELECT 1;\n    SELECT 2;",
			expected: []struct{ start, end Location }{
				{Location{Pos: 2, Line: 1, Col: 3}, Location{Pos: 11, Line: 1, Col: 12}},
				{Location{Pos: 16, Line: 2, Col: 5}, Location{Pos: 25, Line: 2, Col: 14}},
			},
		},
		{
			name:  "multiline statement",
			input: "SELECT\n  1;",
			expected: []struct{ start, end Location }{
				{Location{Pos: 0, Line: 1, Col: 1}, Location{Pos: 11, Line: 2, Col: 5}},
			},
		},
		{
			name:  "statement after multiline comment",
			input: "/* comment\n   */ SELECT 1;",
			expected: []struct{ start, end Location }{
				{Location{Pos: 0, Line: 1, Col: 1}, Location{Pos: 26, Line: 2, Col: 16}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stmts, err := Split(tt.input)
			require.NoError(t, err)
			require.Len(t, stmts, len(tt.expected))

			for i, stmt := range stmts {
				exp := tt.expected[i]
				require.Equal(t, exp.start, stmt.Start, "stmt %d: start", i)
				require.Equal(t, exp.end, stmt.End, "stmt %d: end", i)
			}
		})
	}
}

func TestUnclosedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "unclosed string",
			input:       "SELECT 'unclosed",
			errContains: "unclosed string starting at 1:8",
		},
		{
			name:        "unclosed string on line 2",
			input:       "SELECT 1;\nSELECT 'unclosed",
			errContains: "unclosed string starting at 2:8",
		},
		{
			name:        "unclosed block comment",
			input:       "SELECT /* unclosed",
			errContains: "unclosed block comment starting at 1:8",
		},
		{
			name:        "unclosed nested block comment",
			input:       "SELECT /* outer /* inner */",
			errContains: "unclosed block comment starting at 1:8",
		},
		{
			name:        "unclosed dollar string",
			input:       "SELECT $$unclosed",
			errContains: "unclosed dollar-quoted string starting at 1:8",
		},
		{
			name:        "unclosed dollar string with tag",
			input:       "SELECT $func$unclosed",
			errContains: "unclosed dollar-quoted string $func$ starting at 1:8",
		},
		{
			name:        "unclosed quoted identifier",
			input:       `SELECT "unclosed`,
			errContains: "unclosed quoted identifier starting at 1:8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Split(tt.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errContains,
				"expected error containing %q, got %q", tt.errContains, err.Error())
		})
	}
}
