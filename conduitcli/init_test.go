package conduitcli

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/timegenerator"
)

//nolint:gochecknoglobals
var initTimeGen = timegenerator.Stub{T: time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)}

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("should create migration directory and schema file, when initialising fresh project", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitArgs{
			Dir:         dir,
			DatabaseURL: databaseURL,
		}

		err := Init(t.Context(), fs, initTimeGen, args)

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, dir)
	})
}
