package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/internal/testregistry"
	"go.inout.gg/conduit/internal/testutil"
)

func TestApply(t *testing.T) {
	t.Parallel()

	t.Run("should apply migrations up, when valid migrations are provided", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r))

		args := ApplyArgs{
			DatabaseURL:          testutil.ConnString(pool),
			Direction:            direction.DirectionUp,
			SkipSchemaDriftCheck: true,
		}

		err := apply(t.Context(), m, args)

		require.NoError(t, err)
		assert.True(t, testutil.TableExists(t, pool, "users"))
	})

	t.Run("should roll back migration, when direction is down", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r))

		// First apply up.
		err := apply(t.Context(), m, ApplyArgs{
			DatabaseURL:          testutil.ConnString(pool),
			Direction:            direction.DirectionUp,
			SkipSchemaDriftCheck: true,
		})
		require.NoError(t, err)
		require.True(t, testutil.TableExists(t, pool, "users"))

		// Then apply down.
		err = apply(t.Context(), m, ApplyArgs{
			DatabaseURL:          testutil.ConnString(pool),
			Direction:            direction.DirectionDown,
			SkipSchemaDriftCheck: true,
		})

		require.NoError(t, err)
		assert.False(t, testutil.TableExists(t, pool, "users"))
	})

	t.Run("should return error, when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql": "CREATE TABLE users (id INT);",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r))

		args := ApplyArgs{
			DatabaseURL: "postgres://invalid:5432/nonexistent",
			Direction:   direction.DirectionUp,
		}

		err := apply(t.Context(), m, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to connect to database")
	})

	t.Run("should return error with hint, when hazard is detected", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql": "---- disable-tx ----\n---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_test (id INT);",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r))

		args := ApplyArgs{
			DatabaseURL:          testutil.ConnString(pool),
			Direction:            direction.DirectionUp,
			SkipSchemaDriftCheck: true,
		}

		err := apply(t.Context(), m, args)

		require.Error(t, err)
		require.ErrorIs(t, err, conduit.ErrHazardDetected)
		assert.ErrorContains(t, err, "use --allow-hazards to proceed")
	})
}
