package sqlsplit

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
)

func TestSplitMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "up only single statement",
			input: "CREATE TABLE users (id int);",
		},
		{
			name: "up only multiple statements",
			input: `CREATE TABLE users (id int);
CREATE TABLE posts (id int);`,
		},
		{
			name: "up and down single statements",
			input: `CREATE TABLE users (id int);
---- create above / drop below ----
DROP TABLE users;`,
		},
		{
			name: "up and down multiple statements",
			input: `CREATE TABLE users (id int);
CREATE TABLE posts (id int);
---- create above / drop below ----
DROP TABLE posts;
DROP TABLE users;`,
		},
		{
			name: "up and down with comments",
			input: `-- Create users table
CREATE TABLE users (id int);
---- create above / drop below ----
-- Drop users table
DROP TABLE users;`,
		},
		{
			name: "up and down with dollar-quoted function",
			input: `CREATE FUNCTION test() RETURNS int AS $$
BEGIN
    RETURN 1;
END;
$$ LANGUAGE plpgsql;
---- create above / drop below ----
DROP FUNCTION test();`,
		},
		{
			name: "empty up section",
			input: `---- create above / drop below ----
DROP TABLE users;`,
		},
		{
			name: "empty down section",
			input: `CREATE TABLE users (id int);
---- create above / drop below ----`,
		},
		{
			name:  "separator in string is not split",
			input: `SELECT '---- create above / drop below ----';`,
		},
		{
			name: "disable-tx directive",
			input: `---- disable-tx ----
CREATE TABLE users (id int);
---- create above / drop below ----
DROP TABLE users;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			up, down, err := SplitMigration(tt.input)
			require.NoError(t, err)
			snaps.MatchSnapshot(t, up, down)
		})
	}
}

func TestSplitMigrationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name: "multiple separators",
			input: `CREATE TABLE users (id int);
---- create above / drop below ----
DROP TABLE users;
---- create above / drop below ----
DROP TABLE posts;`,
			errContains: "multiple separators found",
		},
		{
			name:        "unclosed string in up section",
			input:       `SELECT 'unclosed`,
			errContains: "unclosed string",
		},
		{
			name: "unclosed string in down section",
			input: `CREATE TABLE users (id int);
---- create above / drop below ----
SELECT 'unclosed`,
			errContains: "unclosed string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := SplitMigration(tt.input)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errContains)
		})
	}
}
