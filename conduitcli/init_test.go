package conduitcli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/hashsum"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("should create migration directory and schema file, when initialising fresh project", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).Build()
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := InitArgs{
			RootDir:       baseDir,
			ConfigName:    "conduit.yaml",
			MigrationsDir: migrationsDir,
			DatabaseURL:   databaseURL,
		}

		err := Init(t.Context(), fs, timeGen, store, args)

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, baseDir)
	})
}
