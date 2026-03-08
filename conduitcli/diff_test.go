package conduitcli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("should return error, when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       "/",
			MigrationsDir: "/nonexistent",
			Name:          "add_posts",
			SchemaPath:    "/schema.sql",
			DatabaseURL:   "postgres://localhost:5432/testdb",
		}

		_, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrMigrationsNotFound)
		snaps.MatchSnapshot(t, err.Error())
	})

	t.Run("should return error, when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).Build()
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    "/schema.sql",
			DatabaseURL:   "://invalid",
		}

		_, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse database URL")
	})

	t.Run("should create migration file, when schema has new table", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}

		result, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.NoError(t, err)
		require.Len(t, result.Files, 1)
		testutil.SnapshotFS(t, fs, baseDir)
	})

	t.Run("should return error, when source schema hash does not match conduit.sum", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("conduit.sum", "0000000000000000").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}

		_, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.Error(t, err)
		require.ErrorIs(t, err, conduit.ErrSchemaDrift)

		// No migration file should have been created.
		entries, _ := afero.ReadDir(fs, dir)
		for _, e := range entries {
			assert.NotContains(t, e.Name(), "add_posts",
				"migration file should not be created on drift")
		}
	})

	t.Run("should succeed, when source schema hash matches conduit.sum", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// First run to generate the correct conduit.sum.
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}
		_, err := Diff(t.Context(), fs, timeGen, bi, store, args)
		require.NoError(t, err)

		// Update schema to trigger a new diff, using the existing conduit.sum
		// which now contains the correct target hash from the first run.
		require.NoError(
			t,
			afero.WriteFile(fs, filepath.Join(baseDir, "schema.sql"), []byte(`CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);
CREATE TABLE comments (id int, post_id int);`), 0o644),
		)

		args2 := DiffArgs{
			MigrationsDir: dir,
			Name:          "add_comments",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}

		// Act — second diff should succeed because the source hash matches conduit.sum.
		_, err = Diff(t.Context(), fs, timegenerator.Stub{
			T: time.Date(2024, 2, 15, 12, 30, 45, 0, time.UTC),
		}, bi, store, args2)

		require.NoError(t, err)
	})

	t.Run("should return no changes, when schema is already in sync", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", "CREATE TABLE users (id int);").
			Build()

		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "no_changes",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}

		result, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.ErrorIs(t, err, ErrNoChanges)
		assert.Nil(t, result)
	})
}

func TestDiffCreatesFilePerStatement(t *testing.T) {
	t.Parallel()

	t.Run("should create one file per statement, when diff contains multiple DDL", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);
CREATE INDEX idx_posts_user_id ON posts (user_id);`).
			Build()

		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    filepath.Join(baseDir, "schema.sql"),
			DatabaseURL:   databaseURL,
		}

		result, err := Diff(t.Context(), fs, timeGen, bi, store, args)

		require.NoError(t, err)
		require.Len(t, result.Files, 2)
		testutil.SnapshotFS(t, fs, baseDir)
	})
}
