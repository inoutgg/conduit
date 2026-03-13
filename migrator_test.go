package conduit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/testregistry"
	"go.inout.gg/conduit/internal/testutil"
)

func newConn(t *testing.T) (*pgxpool.Pool, *pgx.Conn) {
	t.Helper()

	pool := poolFactory.Pool(t)

	conn, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	return pool, conn.Conn()
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

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// Act
		seq, err := m.Migrate(
			t.Context(),
			conduit.DirectionUp,
			conn,
			nil,
		)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 2)
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

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// Act
		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)
		testutil.CollectSeq2(t, seq)

		// Run migration again
		seq, err = m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Empty(t, results)
	})

	t.Run("should apply only one migration, when step count is 1", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_a.up.sql": "CREATE TABLE a (id INT);",
			"20230602120000_create_b.up.sql": "CREATE TABLE b (id INT);",
			"20230603120000_create_c.up.sql": "CREATE TABLE c (id INT);",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r))

		// Act
		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, &conduit.MigrateOptions{Steps: 1})

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 1)
		assert.Equal(t, "create_a", results[0].Name)
		assert.Equal(t, []dbsqlc.TestAllMigrationsRow{
			{Version: "20230601120000", Name: "create_a"},
		}, appliedMigrations(t, pool))
	})

	t.Run("should apply migration, when running in non-tx mode", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_nontx.up.sql":   "CREATE TABLE nontx_test (id INT);",
			"20230601120000_create_nontx.down.sql": "DROP TABLE nontx_test;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// Act
		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 1)
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

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)
		testutil.CollectSeq2(t, seq)

		// Act
		seq, err = m.Migrate(t.Context(), conduit.DirectionDown, conn, nil)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 1)
		assert.Equal(t, "create_posts", results[0].Name)
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

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_users.up.sql":   "CREATE TABLE users (id INT);",
			"20230601120000_create_users.down.sql": "DROP TABLE users;",
			"20230602120000_create_posts.up.sql":   "CREATE TABLE posts (id INT);",
			"20230602120000_create_posts.down.sql": "DROP TABLE posts;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)
		testutil.CollectSeq2(t, seq)

		// Act
		seq, err = m.Migrate(t.Context(), conduit.DirectionDown, conn, &conduit.MigrateOptions{
			Steps: conduit.AllSteps,
		})

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 2)
		assert.False(t, testutil.TableExists(t, pool, "users"))
		assert.False(t, testutil.TableExists(t, pool, "posts"))
		assert.Empty(t, appliedMigrations(t, pool))
	})

	t.Run("should roll back migration, when running in non-tx mode", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_create_nontx.up.sql":   "CREATE TABLE nontx_test (id INT);",
			"20230601120000_create_nontx.down.sql": "DROP TABLE nontx_test;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)
		testutil.CollectSeq2(t, seq)

		// Act
		seq, err = m.Migrate(t.Context(), conduit.DirectionDown, conn, nil)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 1)
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

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql":   "---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_test (id INT);",
			"20230601120000_hazardous.down.sql": "DROP TABLE hazard_test;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// Act
		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)

		// Assert
		require.NoError(t, err)
		iterErr := testutil.CollectSeq2Error(t, seq)
		require.ErrorIs(t, iterErr, conduit.ErrHazardDetected)
		assert.False(t, testutil.TableExists(t, pool, "hazard_test"))
	})

	t.Run("should allow migration, when AllowHazards is set", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool, conn := newConn(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20230601120000_hazardous.up.sql":   "---- hazard: INDEX_BUILD // rebuilds index ----\nCREATE TABLE hazard_allowed (id INT);",
			"20230601120000_hazardous.down.sql": "DROP TABLE hazard_allowed;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		// Act
		seq, err := m.Migrate(
			t.Context(),
			conduit.DirectionUp,
			conn,
			&conduit.MigrateOptions{
				AllowHazards: []conduit.HazardType{conduit.HazardTypeIndexBuild},
			},
		)

		// Assert
		require.NoError(t, err)
		results := testutil.CollectSeq2(t, seq)
		assert.Len(t, results, 1)
		assert.True(t, testutil.TableExists(t, pool, "hazard_allowed"))
	})
}

func TestMigrator_Migrate_Ordering(t *testing.T) {
	t.Parallel()

	t.Run("should roll up/back same-version migrations in correct order", func(t *testing.T) {
		t.Parallel()

		_, conn := newConn(t)

		r := testregistry.NewRegistry(t, map[string]string{
			"20240115123045_add_posts_01.up.sql":   "CREATE TABLE add_posts_down_01 (id INT);",
			"20240115123045_add_posts_01.down.sql": "DROP TABLE add_posts_down_01;",
			"20240115123045_add_posts_02.up.sql":   "CREATE TABLE add_posts_down_02 (id INT);",
			"20240115123045_add_posts_02.down.sql": "DROP TABLE add_posts_down_02;",
			"20240115123045_add_posts_03.up.sql":   "CREATE TABLE add_posts_down_03 (id INT);",
			"20240115123045_add_posts_03.down.sql": "DROP TABLE add_posts_down_03;",
			"20240115123045_add_posts_04.up.sql":   "CREATE TABLE add_posts_down_04 (id INT);",
			"20240115123045_add_posts_04.down.sql": "DROP TABLE add_posts_down_04;",
			"20240115123045_add_posts_05.up.sql":   "CREATE TABLE add_posts_down_05 (id INT);",
			"20240115123045_add_posts_05.down.sql": "DROP TABLE add_posts_down_05;",
			"20240115123045_add_posts_06.up.sql":   "CREATE TABLE add_posts_down_06 (id INT);",
			"20240115123045_add_posts_06.down.sql": "DROP TABLE add_posts_down_06;",
			"20240115123045_add_posts_07.up.sql":   "CREATE TABLE add_posts_down_07 (id INT);",
			"20240115123045_add_posts_07.down.sql": "DROP TABLE add_posts_down_07;",
			"20240115123045_add_posts_08.up.sql":   "CREATE TABLE add_posts_down_08 (id INT);",
			"20240115123045_add_posts_08.down.sql": "DROP TABLE add_posts_down_08;",
			"20240115123045_add_posts_09.up.sql":   "CREATE TABLE add_posts_down_09 (id INT);",
			"20240115123045_add_posts_09.down.sql": "DROP TABLE add_posts_down_09;",
			"20240115123045_add_posts_10.up.sql":   "CREATE TABLE add_posts_down_10 (id INT);",
			"20240115123045_add_posts_10.down.sql": "DROP TABLE add_posts_down_10;",
		})
		m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

		seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
		require.NoError(t, err)

		results := testutil.CollectSeq2(t, seq)

		require.Len(t, results, 10)

		for i, res := range results {
			assert.Equal(t, fmt.Sprintf("add_posts_%02d", i+1), res.Name)
		}

		seq, err = m.Migrate(t.Context(), conduit.DirectionDown, conn, &conduit.MigrateOptions{
			Steps: conduit.AllSteps,
		})
		require.NoError(t, err)
		results = testutil.CollectSeq2(t, seq)

		require.Len(t, results, 10)

		for i, res := range results {
			assert.Equal(t, fmt.Sprintf("add_posts_%02d", 10-i), res.Name)
		}
	})
}

func TestMigrator_Migrate_Result(t *testing.T) {
	t.Parallel()

	// Arrange
	_, conn := newConn(t)

	r := testregistry.NewRegistry(t, map[string]string{
		"20230603120000_create_c.up.sql":   "CREATE TABLE c_result (id INT);",
		"20230603120000_create_c.down.sql": "DROP TABLE c_result;",
		"20230601120000_create_a.up.sql":   "CREATE TABLE a_result (id INT);",
		"20230601120000_create_a.down.sql": "DROP TABLE a_result;",
		"20230602120000_create_b.up.sql":   "CREATE TABLE b_result (id INT);",
		"20230602120000_create_b.down.sql": "DROP TABLE b_result;",
	})
	m := conduit.NewMigrator(conduit.WithRegistry(r), conduit.WithSkipSchemaDriftCheck())

	// Act — up
	seq, err := m.Migrate(t.Context(), conduit.DirectionUp, conn, nil)
	require.NoError(t, err)
	upResults := testutil.CollectSeq2(t, seq)

	// Assert — ascending order
	require.Len(t, upResults, 3)
	assert.Equal(t, "create_a", upResults[0].Name)
	assert.Equal(t, "create_b", upResults[1].Name)
	assert.Equal(t, "create_c", upResults[2].Name)

	// Assert — positive duration
	for _, mr := range upResults {
		assert.Greater(t, mr.DurationTotal, time.Duration(0))
	}

	// Act — down all
	seq, err = m.Migrate(t.Context(), conduit.DirectionDown, conn, &conduit.MigrateOptions{
		Steps: conduit.AllSteps,
	})
	require.NoError(t, err)
	downResults := testutil.CollectSeq2(t, seq)

	// Assert — descending order
	require.Len(t, downResults, 3)
	assert.Equal(t, "create_c", downResults[0].Name)
	assert.Equal(t, "create_b", downResults[1].Name)
	assert.Equal(t, "create_a", downResults[2].Name)
}
