package initialise

import (
	"os"
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

	t.Run("creates migration directory and initial schema migration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := InitialiseArgs{
			Dir:         dir,
			DatabaseURL: databaseURL,
		}

		// Act
		err := initialise(t.Context(), fs, timeGen, args)

		// Assert
		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, dir)
	})
}
