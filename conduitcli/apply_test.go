package conduitcli

import (
	"bytes"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
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
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		args := ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionUp,
		}

		err := Apply(t.Context(), m, args)

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
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// First apply up.
		err := Apply(t.Context(), m, ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionUp,
		})
		require.NoError(t, err)
		require.True(t, testutil.TableExists(t, pool, "users"))

		// Then apply down.
		err = Apply(t.Context(), m, ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionDown,
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

		err := Apply(t.Context(), m, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to connect to database")
	})

	t.Run("should return error, when hazard is detected", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql": "---- disable-tx ----\n---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_test (id INT);",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		args := ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionUp,
		}

		err := Apply(t.Context(), m, args)

		require.Error(t, err)
		require.ErrorIs(t, err, conduit.ErrHazardDetected)
		snaps.MatchSnapshot(t, err.Error())
	})

	t.Run("should log all pending migrations without applying, when dry-run is enabled", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":    "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql":  "DROP TABLE users;",
			"20230602120000_create_orders.up.sql":   "CREATE TABLE orders (id INT);",
			"20230602120000_create_orders.down.sql": "DROP TABLE orders;",
			"20230603120000_create_items.up.sql":    "CREATE TABLE items (id INT);",
			"20230603120000_create_items.down.sql":  "DROP TABLE items;",
		})

		var buf bytes.Buffer

		m := conduit.NewMigrator(
			conduit.WithRegistry(r),
			conduit.WithExecutor(conduit.NewDryRunExecutor(&buf, false)),
			conduit.WithSkipSchemaDriftCheck(),
		)

		err := Apply(t.Context(), m, ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionUp,
		})

		require.NoError(t, err)
		assert.False(t, testutil.TableExists(t, pool, "users"))
		assert.False(t, testutil.TableExists(t, pool, "orders"))
		assert.False(t, testutil.TableExists(t, pool, "items"))
		snaps.MatchSnapshot(t, buf.String())
	})

	t.Run(
		"should list all applied migrations for rollback without dropping, when dry-run is enabled with direction down",
		func(t *testing.T) {
			t.Parallel()

			pool := poolFactory.Pool(t)

			r := testregistry.NewRegistry(t, map[string]string{
				"20230601120000_create_users.up.sql":    "CREATE TABLE users (id INT);",
				"20230601120000_create_users.down.sql":  "DROP TABLE users;",
				"20230602120000_create_orders.up.sql":   "CREATE TABLE orders (id INT);",
				"20230602120000_create_orders.down.sql": "DROP TABLE orders;",
			})

			// First apply all up for real.
			m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

			err := Apply(t.Context(), m, ApplyArgs{
				DatabaseURL: testutil.ConnString(pool),
				Direction:   direction.DirectionUp,
			})
			require.NoError(t, err)
			require.True(t, testutil.TableExists(t, pool, "users"))
			require.True(t, testutil.TableExists(t, pool, "orders"))

			// Dry-run down should list migration but not drop the tables.
			// Default down step is 1, so only the latest migration should be listed.
			var buf bytes.Buffer

			dryRunMigrator := conduit.NewMigrator(
				conduit.WithRegistry(r),
				conduit.WithExecutor(conduit.NewDryRunExecutor(&buf, false)),
				conduit.WithSkipSchemaDriftCheck(),
			)

			err = Apply(t.Context(), dryRunMigrator, ApplyArgs{
				DatabaseURL: testutil.ConnString(pool),
				Direction:   direction.DirectionDown,
			})

			require.NoError(t, err)
			assert.True(t, testutil.TableExists(t, pool, "users"))
			assert.True(t, testutil.TableExists(t, pool, "orders"))
			snaps.MatchSnapshot(t, buf.String())
		},
	)

	t.Run("should include SQL content, when dry-run is enabled with verbose", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_orders.up.sql":  "CREATE TABLE orders (id INT);\nCREATE INDEX idx_orders_id ON orders (id);",
		})

		var buf bytes.Buffer

		m := conduit.NewMigrator(
			conduit.WithRegistry(r),
			conduit.WithExecutor(conduit.NewDryRunExecutor(&buf, true)),
			conduit.WithSkipSchemaDriftCheck(),
		)

		err := Apply(t.Context(), m, ApplyArgs{
			DatabaseURL: testutil.ConnString(pool),
			Direction:   direction.DirectionUp,
		})

		require.NoError(t, err)
		assert.False(t, testutil.TableExists(t, pool, "users"))
		assert.False(t, testutil.TableExists(t, pool, "orders"))
		snaps.MatchSnapshot(t, buf.String())
	})
}
