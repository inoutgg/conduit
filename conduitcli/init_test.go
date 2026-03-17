package conduitcli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/lockfile"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("should create migration directory and schema file, when initialising fresh project", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).Build()
		store := lockfile.NewFSStore(fs, "conduit.lock")
		args := InitArgs{
			RootDir:       baseDir,
			ConfigName:    "conduit.yaml",
			MigrationsDir: migrationsDir,
			DatabaseURL:   databaseURL,
		}

		result, err := Init(t.Context(), fs, timeGen, store, args)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, migrationsDir, result.MigrationsDirPath)
		assert.NotEmpty(t, result.MigrationPath)
		assert.Equal(t, "conduit.yaml", result.ConfigPath)
		assert.Equal(t, "conduit.lock", result.LockfilePath)
		testutil.SnapshotFS(t, fs, baseDir)
	})
}
