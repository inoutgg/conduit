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

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
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

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
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

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql",
				"---- disable-tx ----\nCREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql",
				"---- disable-tx ----\nDROP TABLE users;").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		m := migrations[0]

		upTx, err := m.UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.False(t, upTx, "disable-tx directive should disable transactions for up migration")

		downTx, err := m.UseTx(direction.DirectionDown)
		require.NoError(t, err)
		assert.False(t, downTx, "disable-tx directive should disable transactions for down migration")
	})

	t.Run("disable-tx directive only in up migration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.up.sql",
				"---- disable-tx ----\nCREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql",
				"DROP TABLE users;").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
		require.NoError(t, err)
		require.Len(t, migrations, 1)

		m := migrations[0]

		upTx, err := m.UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.False(t, upTx, "up migration with disable-tx directive should not use transactions")

		downTx, err := m.UseTx(direction.DirectionDown)
		require.NoError(t, err)
		assert.True(t, downTx, "down migration without disable-tx directive should use transactions")
	})

	t.Run("multiple versions sorted", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601130000_add_email.up.sql",
				"ALTER TABLE users ADD COLUMN email TEXT;").
			WithFile("20230601120000_create_user.up.sql", "CREATE TABLE users (id INT);").
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
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

	t.Run("returns error on SQL file without direction suffix", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.sql", "CREATE TABLE users (id INT);").
			Build()

		// Act
		_, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must have .up.sql or .down.sql suffix")
	})

	t.Run("returns error on down-only migration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_create_user.down.sql", "DROP TABLE users;").
			Build()

		// Act
		_, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has a down file but no up file")
	})

	t.Run("same version different names coexist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_add_posts_1.up.sql", "CREATE TABLE posts (id INT);").
			WithFile("20230601120000_add_posts_2.up.sql",
				"---- disable-tx ----\nCREATE INDEX CONCURRENTLY idx ON posts (id);").
			Build()

		// Act
		migrations, err := parseSQLMigrationsFromFS(fs, dir)

		// Assert
		require.NoError(t, err)
		require.Len(t, migrations, 2)

		slices.SortFunc(migrations, func(a, b *Migration) int {
			if c := a.Version().Compare(b.Version()); c != 0 {
				return c
			}

			if a.Name() < b.Name() {
				return -1
			}

			return 1
		})

		assert.Equal(t, "add_posts_1", migrations[0].Name())
		assert.Equal(t, "add_posts_2", migrations[1].Name())
		assert.Equal(t, migrations[0].Version().String(), migrations[1].Version().String())

		upTx1, err := migrations[0].UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.True(t, upTx1)

		upTx2, err := migrations[1].UseTx(direction.DirectionUp)
		require.NoError(t, err)
		assert.False(t, upTx2)
	})
}
