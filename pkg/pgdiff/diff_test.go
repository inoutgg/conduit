package pgdiff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
)

func TestReadStmtsFromFile(t *testing.T) {
	t.Parallel()

	t.Run("reads and parses multiple statements", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);
CREATE INDEX idx_posts ON posts (id);`).
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "schema.sql"))

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()

		// Act
		_, err := readStmtsFromFile(fs, filepath.Join(dir, "nonexistent.sql"))

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read file")
	})

	t.Run("returns error on invalid SQL", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("bad.sql", "SELECT 'unclosed string").
			Build()

		// Act
		_, err := readStmtsFromFile(fs, filepath.Join(dir, "bad.sql"))

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "unclosed string")
	})

	t.Run("returns empty slice for empty file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("empty.sql", "").
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "empty.sql"))

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("reads all statements from file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);`).
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "schema.sql"))

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})
}

func TestGeneratePlan(t *testing.T) {
	t.Parallel()

	t.Run("generates plan with new table", func(t *testing.T) {
		t.Parallel()

		// Arrange
		config, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
		require.NoError(t, err)

		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// Act
		plan, err := GeneratePlan(
			t.Context(),
			fs,
			config,
			migrationsDir,
			filepath.Join(baseDir, "schema.sql"),
		)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, plan.CurrentSchemaHash, plan.Statements)
	})
}

func TestReadStmtsFromMigrationsDir(t *testing.T) {
	t.Parallel()

	t.Run("sorts files by version timestamp", func(t *testing.T) {
		t.Parallel()

		// Arrange
		// Migration filenames use YYYYMMDDHHMMSS format
		// Files added out of order to verify sorting
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601140000_third.up.sql", "CREATE TABLE third (id int);").
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			WithFile("20230601130000_second.up.sql", "CREATE TABLE second (id int);").
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithSubdir("subdir.sql"). // tricky: dir ending in .sql
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			WithReadError("20230601120000_first.up.sql", os.ErrPermission).
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_bad.up.sql", "SELECT 'unclosed string").
			Build()

		// Act
		_, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "unclosed string")
	})

	t.Run("returns error on invalid migration filename", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("returns empty slice for directory with only non-sql files", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("README.md", "# Migrations").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("skips down migration files", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_users.up.sql", "CREATE TABLE users (id int);").
			WithFile("20230601120000_users.down.sql", "DROP TABLE users;").
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", `CREATE TABLE users (id int);
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
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE a (id int); CREATE TABLE b (id int);").
			WithFile("20230601130000_second.up.sql", "CREATE TABLE c (id int);").
			Build()

		// Act
		stmts, err := readStmtsFromMigrationsDir(fs, dir)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})
}
