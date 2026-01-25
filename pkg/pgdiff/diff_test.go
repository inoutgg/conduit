package pgdiff

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadStmtsFromMigrationsDir(t *testing.T) {
	t.Parallel()

	t.Run("sorts files by version timestamp", func(t *testing.T) {
		t.Parallel()

		// Arrange
		// Migration filenames use YYYYMMDDHHMMSS format
		// Files added out of order to verify sorting
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601140000_third.sql", "CREATE TABLE third (id int);").
			WithFile("20230601120000_first.sql", "CREATE TABLE first (id int);").
			WithFile("20230601130000_second.sql", "CREATE TABLE second (id int);").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("skips non-sql files", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_first.sql", "CREATE TABLE first (id int);").
			WithFile("README.md", "# Migrations").
			WithFile("config.json", "{}").
			WithFile(".gitkeep", "").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithSubdir("subdir.sql"). // tricky: dir ending in .sql
			WithFile("20230601120000_first.sql", "CREATE TABLE first (id int);").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("returns error when directory does not exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs := afero.NewMemMapFs()

		// Act
		_, err := readStmtsFromMigrationsDir(fs, "/nonexistent")

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read directory")
	})

	t.Run("returns error when file cannot be read", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_first.sql", "CREATE TABLE first (id int);").
			WithReadError("20230601120000_first.sql", os.ErrPermission).
			Build()

		// Act
		_, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read file")
	})

	t.Run("returns error on invalid SQL", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_bad.sql", "NOT VALID SQL {{{{").
			Build()

		// Act
		_, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse SQL")
	})

	t.Run("returns error on invalid migration filename", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("invalid_filename.sql", "CREATE TABLE test (id int);").
			Build()

		// Act
		_, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse migration filename")
	})

	t.Run("returns empty slice for empty directory", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("returns empty slice for directory with only non-sql files", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("README.md", "# Migrations").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("only uses up statements ignoring down section", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_users.sql", `CREATE TABLE users (id int);
---- create above / drop below ----
DROP TABLE users;`).
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("handles multiple statements per file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_init.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);
CREATE INDEX idx_posts ON posts (id);`).
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("aggregates statements from multiple files in order", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, dir := newMigrationsBuilder(t).
			WithFile("20230601120000_first.sql", "CREATE TABLE a (id int); CREATE TABLE b (id int);").
			WithFile("20230601130000_second.sql", "CREATE TABLE c (id int);").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})
}
