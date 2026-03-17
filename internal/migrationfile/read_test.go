package migrationfile

import (
	"os"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
)

func TestReadMigrationsFromDir(t *testing.T) {
	t.Parallel()

	t.Run("should sort files by version, when added out of order", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601140000_third.up.sql", "CREATE TABLE third (id int);").
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			WithFile("20230601130000_second.up.sql", "CREATE TABLE second (id int);").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})

	t.Run("should skip non-sql files, when directory contains mixed file types", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			WithFile("README.md", "# Migrations").
			WithFile("config.json", "{}").
			WithFile(".gitkeep", "").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})

	t.Run("should skip subdirectories, when directory name ends in .sql", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithSubdir("subdir.sql").
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})

	t.Run("should return error, when directory does not exist", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()

		_, err := ReadMigrationsFromDir(fs, "/nonexistent")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read directory")
	})

	t.Run("should return error, when file cannot be read", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE first (id int);").
			WithReadError("20230601120000_first.up.sql", os.ErrPermission).
			Build()

		_, err := ReadMigrationsFromDir(fs, dir)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read file")
	})

	t.Run("should return error, when migration SQL is invalid", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_bad.up.sql", "SELECT 'unclosed string").
			Build()

		_, err := ReadMigrationsFromDir(fs, dir)

		require.Error(t, err)
		assert.ErrorContains(t, err, "unclosed string")
	})

	t.Run("should return error, when migration filename is invalid", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("invalid_filename.sql", "CREATE TABLE test (id int);").
			Build()

		_, err := ReadMigrationsFromDir(fs, dir)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse migration filename")
	})

	t.Run("should return empty slice, when directory is empty", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		assert.Empty(t, migrations)
	})

	t.Run("should return empty slice, when directory contains only non-sql files", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("README.md", "# Migrations").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		assert.Empty(t, migrations)
	})

	t.Run("should skip down files, when both up and down migrations exist", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_users.up.sql", "CREATE TABLE users (id int);").
			WithFile("20230601120000_users.down.sql", "DROP TABLE users;").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})

	t.Run("should parse all statements, when file contains multiple DDL", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);
CREATE INDEX idx_posts ON posts (id);`).
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})

	t.Run("should aggregate statements in order, when multiple files exist", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_first.up.sql", "CREATE TABLE a (id int); CREATE TABLE b (id int);").
			WithFile("20230601130000_second.up.sql", "CREATE TABLE c (id int);").
			Build()

		migrations, err := ReadMigrationsFromDir(fs, dir)

		require.NoError(t, err)
		snaps.MatchSnapshot(t, migrations)
	})
}
