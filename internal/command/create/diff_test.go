package create

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/internal/timegenerator"
)

//nolint:gochecknoglobals
var timeGen = timegenerator.Stub{T: time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)}

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("returns error when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs := afero.NewMemMapFs()
		args := DiffArgs{
			Dir:         "/nonexistent",
			Name:        "add_posts",
			SchemaPath:  "/schema.sql",
			DatabaseURL: "postgres://localhost:5432/testdb",
		}

		// Act
		err := diff(t.Context(), fs, timeGen, args)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "migrations directory does not exist")
	})

	t.Run("returns error when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  "/schema.sql",
			DatabaseURL: "://invalid",
		}

		// Act
		err := diff(t.Context(), fs, timeGen, args)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse database URL")
	})

	t.Run("creates migration file from schema diff", func(t *testing.T) {
		t.Parallel()

		// Arrange
		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		// Act
		err := diff(t.Context(), fs, timeGen, args)

		// Assert
		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("returns error when source schema hash does not match conduit.sum", func(t *testing.T) {
		t.Parallel()

		// Arrange
		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithFile("conduit.sum", "0000000000000000").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		// Act
		err := diff(t.Context(), fs, timeGen, args)

		// Assert
		require.Error(t, err)
		require.ErrorContains(t, err, "source schema drift detected")

		// No migration file should have been created.
		entries, _ := afero.ReadDir(fs, dir)
		for _, e := range entries {
			assert.NotContains(t, e.Name(), "add_posts",
				"migration file should not be created on drift")
		}
	})

	t.Run("succeeds when source schema hash matches conduit.sum", func(t *testing.T) {
		t.Parallel()

		// Arrange
		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// First run to generate the correct conduit.sum.
		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}
		require.NoError(t, diff(t.Context(), fs, timeGen, args))

		// Update schema to trigger a new diff, using the existing conduit.sum
		// which now contains the correct target hash from the first run.
		require.NoError(
			t,
			afero.WriteFile(fs, filepath.Join(baseDir, "schema.sql"), []byte(`CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);
CREATE TABLE comments (id int, post_id int);`), 0o644),
		)

		args2 := DiffArgs{
			Dir:         dir,
			Name:        "add_comments",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		// Act — second diff should succeed because the source hash matches conduit.sum.
		err := diff(t.Context(), fs, timegenerator.Stub{
			T: time.Date(2024, 2, 15, 12, 30, 45, 0, time.UTC),
		}, args2)

		// Assert
		require.NoError(t, err)
	})

	t.Run("returns error when no schema changes detected", func(t *testing.T) {
		t.Parallel()

		// Arrange
		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", "CREATE TABLE users (id int);").
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "no_changes",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		// Act
		err := diff(t.Context(), fs, timeGen, args)

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "no schema changes detected")
	})
}
