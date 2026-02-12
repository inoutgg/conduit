package create

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
)

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
