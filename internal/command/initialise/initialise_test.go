package initialise

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/internal/timegenerator"
)

//nolint:gochecknoglobals
var timeGen = timegenerator.Stub{T: time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)}

func TestInitialise(t *testing.T) {
	t.Parallel()

	t.Run("creates migration directory and conduit migration file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:                 dir,
			PackageName:         "migrations",
			Namespace:           "",
			NoConduitMigrations: false,
		}

		// Act
		err := initialise(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("creates registry file when namespace is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:                 dir,
			PackageName:         "migrations",
			Namespace:           "custom",
			NoConduitMigrations: false,
		}

		// Act
		err := initialise(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("skips conduit migration file when NoConduitMigrations is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:                 dir,
			PackageName:         "migrations",
			Namespace:           "",
			NoConduitMigrations: true,
		}

		// Act
		err := initialise(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("creates only registry file when namespace is set and NoConduitMigrations is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:                 dir,
			PackageName:         "migrations",
			Namespace:           "custom",
			NoConduitMigrations: true,
		}

		// Act
		err := initialise(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("uses custom package name in conduit migration file", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:                 dir,
			PackageName:         "custompkg",
			Namespace:           "",
			NoConduitMigrations: false,
		}

		// Act
		err := initialise(fs, timeGen, args)
		require.NoError(t, err)

		// Assert
		testutil.SnapshotFS(t, fs, dir)
	})
}
