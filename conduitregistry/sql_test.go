package conduitregistry

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/internal/testutil"
)

func TestParseSQLMigrationsFromFS(t *testing.T) {
	t.Parallel()

	t.Run("split up and down migration files", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		m := migrations[0]
		assert.Equal(t, "create_user", m.Name())
		assert.Equal(t, "20230601120000", m.Version().String())

		upTx, err := m.UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.True(t, upTx, "up migration should use transaction by default")

		downTx, err := m.UseTx(direction.DirectionDown)
		require.NoError(t, err)
		assert.True(t, downTx, "down migration should use transaction by default")
	})

	t.Run("up-only migration", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		m := migrations[0]
		assert.Equal(t, "create_user", m.Name())
		assert.Equal(t, "20230601120000", m.Version().String())

		upTx, err := m.UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.True(t, upTx)

		assert.Equal(t, emptyMigrateFunc, m.down)
	})

	t.Run("disable-tx directive", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql",
				"---- disable-tx ----\nCREATE TABLE users (id INT);").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		m := migrations[0]
		useTx, err := m.UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.False(t, useTx, "disable-tx directive should disable transactions")
	})

	t.Run("multiple versions sorted", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601130000_add_email.up.sql",
				"ALTER TABLE users ADD COLUMN email TEXT;").
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		migrations, err := parseSQLMigrationsFromFS(fs, dir)
		require.NoError(t, err)
		require.Len(t, migrations, 2)

		slices.SortFunc(migrations, func(a, b *Migration) int {
			return a.Version().Compare(b.Version())
		})

		assert.Equal(t, "create_user", migrations[0].Name())
		assert.Equal(t, "20230601120000", migrations[0].Version().String())
		assert.NotEqual(t, emptyMigrateFunc, migrations[0].down)

		assert.Equal(t, "add_email", migrations[1].Name())
		assert.Equal(t, "20230601130000", migrations[1].Version().String())
		assert.Equal(t, emptyMigrateFunc, migrations[1].down)
	})

	t.Run("error on SQL file without direction suffix", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.sql", "CREATE TABLE users (id INT);").
			Build()

		_, err := parseSQLMigrationsFromFS(fs, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must have .up.sql or .down.sql suffix")
	})

	t.Run("error on down-only migration", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		_, err := parseSQLMigrationsFromFS(fs, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has a down file but no up file")
	})
}
