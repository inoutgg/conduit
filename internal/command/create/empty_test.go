package create

import (
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

func TestEmpty(t *testing.T) {
	t.Parallel()

	t.Run("creates SQL migration files", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := EmptyArgs{
			Dir:  dir,
			Name: "add_users",
			Ext:  "sql",
		}

		// Act
		err := empty(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("creates Go migration file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := EmptyArgs{
			Dir:         dir,
			Name:        "add_users",
			Ext:         "go",
			PackageName: "migrations",
		}

		// Act
		err := empty(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("creates Go migration file with custom registry", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("registry.go", "package migrations").
			Build()
		args := EmptyArgs{
			Dir:         dir,
			Name:        "add_users",
			Ext:         "go",
			PackageName: "migrations",
		}

		// Act
		err := empty(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("creates Go migration file with custom package name", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := EmptyArgs{
			Dir:         dir,
			Name:        "add_users",
			Ext:         "go",
			PackageName: "db",
		}

		// Act
		err := empty(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("returns error when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs := afero.NewMemMapFs()
		args := EmptyArgs{
			Dir:  "/nonexistent",
			Name: "add_users",
			Ext:  "sql",
		}

		// Act & Assert
		err := empty(fs, timeGen, args)
		assert.Error(t, err)
	})
}
