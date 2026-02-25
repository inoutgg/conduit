package conduit_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/testutil"
)

func newTestMigrator(t *testing.T, registry *conduitregistry.Registry, opts ...conduit.Option) *conduit.Migrator {
	t.Helper()

	opts = append([]conduit.Option{
		conduit.WithRegistry(registry),
		conduit.WithSkipSchemaDriftCheck(),
	}, opts...)

	return conduit.NewMigrator(conduit.NewConfig(opts...))
}

func newConn(t *testing.T) (*pgxpool.Pool, *pgx.Conn) {
	t.Helper()

	pool := poolFactory.Pool(t)

	conn, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	return pool, conn.Conn()
}

func newRegistry(t *testing.T, files map[string]string) *conduitregistry.Registry {
	t.Helper()

	builder := testutil.NewMigrationsDirBuilder(t)
	for name, content := range files {
		builder.WithFile(name, content)
	}

	fs, _, dir := builder.Build()

	return conduitregistry.FromFS(fs, dir)
}

func appliedMigrations(t *testing.T, pool *pgxpool.Pool) []dbsqlc.TestAllMigrationsRow {
	t.Helper()

	rows, err := dbsqlc.New().TestAllMigrations(t.Context(), pool)
	require.NoError(t, err)

	return rows
}

func TestMigrator_MigrateUp(t *testing.T) {
	t.Parallel()

	t.Run("should apply all migrations, when all are pending", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := newTestMigrator(t, r)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 2)
		assert.Equal(t, conduit.DirectionUp, result.Direction)
		assert.True(t, testutil.TableExists(t, pool, "users"))
		assert.True(t, testutil.TableExists(t, pool, "posts"))
		assert.Equal(t, []dbsqlc.TestAllMigrationsRow{
			{Version: "20230601120000", Name: "create_users"},
			{Version: "20230602120000", Name: "create_posts"},
		}, appliedMigrations(t, pool))
	})

	t.Run("should skip migrations, when already applied", func(t *testing.T) {
		t.Parallel()

		// Arrange
		_, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
		})
		m := newTestMigrator(t, r)

		// Act
		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)

		// Run migration again
		result, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, result.MigrationResults)
	})

	t.Run("should apply only one migration, when step count is 1", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_a.up.sql": "CREATE TABLE a (id INT);",
			"20230602120000_create_b.up.sql": "CREATE TABLE b (id INT);",
			"20230603120000_create_c.up.sql": "CREATE TABLE c (id INT);",
		})
		m := newTestMigrator(t, r)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, &conduit.MigrateOptions{Steps: 1})

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 1)
		assert.Equal(t, "create_a", result.MigrationResults[0].Name)
		assert.Equal(t, []dbsqlc.TestAllMigrationsRow{
			{Version: "20230601120000", Name: "create_a"},
		}, appliedMigrations(t, pool))
	})

	t.Run("should apply migration, when disable-tx directive is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_nontx.up.sql":   "---- disable-tx ----\nCREATE TABLE nontx_test (id INT);",
			"20230601120000_create_nontx.down.sql": "---- disable-tx ----\nDROP TABLE nontx_test;",
		})
		m := newTestMigrator(t, r)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 1)
		assert.True(t, testutil.TableExists(t, pool, "nontx_test"))
		assert.Equal(t, []dbsqlc.TestAllMigrationsRow{
			{Version: "20230601120000", Name: "create_nontx"},
		}, appliedMigrations(t, pool))
	})
}

func TestMigrator_MigrateDown(t *testing.T) {
	t.Parallel()

	t.Run("should roll back one migration, when no step count is given", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := newTestMigrator(t, r)

		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionDown, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 1)
		assert.Equal(t, "create_posts", result.MigrationResults[0].Name)
		assert.False(t, testutil.TableExists(t, pool, "posts"))
		assert.True(t, testutil.TableExists(t, pool, "users"))
		assert.Equal(t, []dbsqlc.TestAllMigrationsRow{
			{Version: "20230601120000", Name: "create_users"},
		}, appliedMigrations(t, pool))
	})

	t.Run("should roll back all migrations, when AllSteps is used", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := newTestMigrator(t, r)

		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionDown, conn, &conduit.MigrateOptions{
			Steps: conduit.AllSteps,
		})

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 2)
		assert.False(t, testutil.TableExists(t, pool, "users"))
		assert.False(t, testutil.TableExists(t, pool, "posts"))
		assert.Empty(t, appliedMigrations(t, pool))
	})

	t.Run("should roll back migration, when disable-tx directive is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_nontx.up.sql":   "---- disable-tx ----\nCREATE TABLE nontx_test (id INT);",
			"20230601120000_create_nontx.down.sql": "---- disable-tx ----\nDROP TABLE nontx_test;",
		})
		m := newTestMigrator(t, r)

		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionDown, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 1)
		assert.False(t, testutil.TableExists(t, pool, "nontx_test"))
		assert.Empty(t, appliedMigrations(t, pool))
	})
}

func TestMigrator_Migrate_Hazards(t *testing.T) {
	t.Parallel()

	t.Run("should block migration, when hazard is detected", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql":   "---- disable-tx ----\n---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_test (id INT);",
			"20230601120000_hazardous.down.sql": "---- disable-tx ----\nDROP TABLE hazard_test;",
		})
		m := newTestMigrator(t, r)

		// Act
		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.ErrorIs(t, err, conduit.ErrHazardDetected)
		assert.False(t, testutil.TableExists(t, pool, "hazard_test"))
	})

	t.Run("should allow migration, when WithAllowHazards is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql":   "---- disable-tx ----\n---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_allowed (id INT);",
			"20230601120000_hazardous.down.sql": "---- disable-tx ----\nDROP TABLE hazard_allowed;",
		})
		m := newTestMigrator(t, r, conduit.WithAllowHazards())

		// Act
		result, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		assert.Len(t, result.MigrationResults, 1)
		assert.True(t, testutil.TableExists(t, pool, "hazard_allowed"))
	})
}

func TestMigrator_Migrate_Result(t *testing.T) {
	t.Parallel()

	// Arrange
	_, conn := newConn(t)

	r := newRegistry(t, map[string]string{
		"20230603120000_create_c.up.sql":   "CREATE TABLE c_result (id INT);",
		"20230603120000_create_c.down.sql": "DROP TABLE c_result;",
		"20230601120000_create_a.up.sql":   "CREATE TABLE a_result (id INT);",
		"20230601120000_create_a.down.sql": "DROP TABLE a_result;",
		"20230602120000_create_b.up.sql":   "CREATE TABLE b_result (id INT);",
		"20230602120000_create_b.down.sql": "DROP TABLE b_result;",
	})
	m := newTestMigrator(t, r)

	// Act — up
	upResult, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
	require.NoError(t, err)

	// Assert — direction
	assert.Equal(t, conduit.DirectionUp, upResult.Direction)

	// Assert — ascending order
	require.Len(t, upResult.MigrationResults, 3)
	assert.Equal(t, "create_a", upResult.MigrationResults[0].Name)
	assert.Equal(t, "create_b", upResult.MigrationResults[1].Name)
	assert.Equal(t, "create_c", upResult.MigrationResults[2].Name)

	// Assert — positive duration
	for _, mr := range upResult.MigrationResults {
		assert.Greater(t, mr.DurationTotal, time.Duration(0))
	}

	// Act — down all
	downResult, err := m.Migrate(t.Context(), conduit.DirectionDown, conn, &conduit.MigrateOptions{
		Steps: conduit.AllSteps,
	})
	require.NoError(t, err)

	// Assert — direction
	assert.Equal(t, conduit.DirectionDown, downResult.Direction)

	// Assert — descending order
	require.Len(t, downResult.MigrationResults, 3)
	assert.Equal(t, "create_c", downResult.MigrationResults[0].Name)
	assert.Equal(t, "create_b", downResult.MigrationResults[1].Name)
	assert.Equal(t, "create_a", downResult.MigrationResults[2].Name)
}

func TestMigrator_Migrate_Validation(t *testing.T) {
	t.Parallel()

	t.Run("should return error, when step count is zero", func(t *testing.T) {
		t.Parallel()

		// Arrange
		_, conn := newConn(t)

		r := newRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql": "CREATE TABLE users (id INT);",
		})
		m := newTestMigrator(t, r)

		// Act
		_, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, &conduit.MigrateOptions{Steps: 0})

		// Assert
		require.ErrorIs(t, err, conduit.ErrInvalidStep)
	})
}
