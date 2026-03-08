package conduitcli

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

		fs, _, migrationsDir := testutil.NewMigrationsDirBuilder(t).Build()
		result, err := New(fs, timeGen, NewArgs{
			MigrationsDir: migrationsDir,
			Name:          "add_users",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, filepath.Join(migrationsDir, "20240115123045_add_users.up.sql"), result.UpFile)
		assert.Equal(t, filepath.Join(migrationsDir, "20240115123045_add_users.down.sql"), result.DownFile)

		upContent, err := afero.ReadFile(fs, result.UpFile)
		require.NoError(t, err)
		assert.Empty(t, upContent)

		downContent, err := afero.ReadFile(fs, result.DownFile)
		require.NoError(t, err)
		assert.Empty(t, downContent)
	})
}
