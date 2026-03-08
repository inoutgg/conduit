package conduitcli

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("should return error when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		_, err := New(fs, timeGen, NewArgs{
			MigrationsDir: "/nonexistent",
			Name:          "add_users",
		})

		require.Error(t, err)
		require.ErrorIs(t, err, ErrMigrationsNotFound)
	})

	t.Run("should create empty up and down migration files", func(t *testing.T) {
		t.Parallel()

		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).Build()
		result, err := New(fs, timeGen, NewArgs{
			MigrationsDir: migrationsDir,
			Name:          "add_users",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		testutil.SnapshotFS(t, fs, baseDir)
	})
}
