package conduitregistry

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/internal/testutil"
)

func TestMigration_Apply(t *testing.T) {
	t.Parallel()

	t.Run("applies up migration", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		ctx := t.Context()

		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		t.Cleanup(conn.Release)

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_test_apply_up.up.sql",
				`---- disable-tx ----
CREATE TABLE test_apply_up (id INT);`).
			WithFile("20230601120000_create_test_apply_up.down.sql",
				`---- disable-tx ----
DROP TABLE test_apply_up;`).
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		err = migrations[0].Apply(ctx, direction.DirectionUp, conn.Conn())
		require.NoError(t, err)

		assert.True(t, testutil.TableExists(t, pool, "test_apply_up"),
			"table should exist after up migration")

		err = migrations[0].Apply(ctx, direction.DirectionDown, conn.Conn())
		require.NoError(t, err)

		assert.False(t, testutil.TableExists(t, pool, "test_apply_up"),
			"table should not exist after down migration")
	})

	t.Run("returns error on invalid SQL", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		ctx := t.Context()

		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		t.Cleanup(conn.Release)

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_bad.up.sql", `---- disable-tx ----
INVALID SQL STATEMENT;`).
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		err = migrations[0].Apply(ctx, direction.DirectionUp, conn.Conn())
		require.Error(t, err)
		snaps.MatchSnapshot(t, err.Error())
	})
}

func TestMigration_ApplyTx(t *testing.T) {
	t.Parallel()

	t.Run("applies up migration in transaction", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		ctx := t.Context()

		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		t.Cleanup(conn.Release)

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_test_apply_tx.up.sql",
				"CREATE TABLE test_apply_tx (id INT);").
			WithFile("20230601120000_create_test_apply_tx.down.sql",
				"DROP TABLE test_apply_tx;").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		err = migrations[0].ApplyTx(ctx, direction.DirectionUp, tx)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		assert.True(t, testutil.TableExists(t, pool, "test_apply_tx"),
			"table should exist after committed tx migration")
	})

	t.Run("returns error on invalid SQL", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		ctx := t.Context()

		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		t.Cleanup(conn.Release)

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_bad_tx.up.sql", "INVALID SQL;").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		err = migrations[0].ApplyTx(ctx, direction.DirectionUp, tx)
		require.Error(t, err)
		snaps.MatchSnapshot(t, err.Error())

		err = tx.Rollback(ctx)
		require.NoError(t, err)
	})
}
