package pgdiff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/internal/testutil"
)

func TestReadStmtsFromFile(t *testing.T) {
	t.Parallel()

	t.Run("should parse all statements, when file contains multiple", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);
CREATE INDEX idx_posts ON posts (id);`).
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "schema.sql"))

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("should return error, when file does not exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()

		// Act
		_, err := readStmtsFromFile(fs, filepath.Join(dir, "nonexistent.sql"))

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read file")
	})

	t.Run("should return error, when SQL is invalid", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("bad.sql", "SELECT 'unclosed string").
			Build()

		// Act
		_, err := readStmtsFromFile(fs, filepath.Join(dir, "bad.sql"))

		// Assert
		require.Error(t, err)
		assert.ErrorContains(t, err, "unclosed string")
	})

	t.Run("should return empty slice, when file is empty", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("empty.sql", "").
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "empty.sql"))

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("should read all statements, when file has two tables", func(t *testing.T) {
		t.Parallel()

		// Arrange
		fs, _, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);`).
			Build()

		// Act
		stmts, err := readStmtsFromFile(fs, filepath.Join(dir, "schema.sql"))

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, stmts)
	})
}

func TestGeneratePlan(t *testing.T) {
	t.Parallel()

	t.Run("should generate plan, when new table is added to schema", func(t *testing.T) {
		t.Parallel()

		// Arrange
		config, err := pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
		require.NoError(t, err)

		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// Act
		plan, err := GeneratePlan(
			t.Context(),
			fs,
			config,
			migrationsDir,
			filepath.Join(baseDir, "schema.sql"),
			nil,
		)

		// Assert
		require.NoError(t, err)
		snaps.MatchSnapshot(t, plan.TargetSchemaHash, plan.SourceSchemaHash, plan.Statements)
	})

	t.Run("should exclude conduit_migrations from plan statements, when schema has changes", func(t *testing.T) {
		t.Parallel()

		// Arrange
		config, err := pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
		require.NoError(t, err)

		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);`).
			Build()

		// Act
		plan, err := GeneratePlan(
			t.Context(),
			fs,
			config,
			migrationsDir,
			filepath.Join(baseDir, "schema.sql"),
			nil,
		)

		// Assert
		require.NoError(t, err)

		for _, stmt := range plan.Statements {
			assert.NotContains(t, stmt.DDL, "conduit_migrations",
				"plan statements should not reference conduit internal tables")
		}
	})

	t.Run("should return empty plan, when source and target schemas are identical", func(t *testing.T) {
		t.Parallel()

		// Arrange
		config, err := pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
		require.NoError(t, err)

		fs, baseDir, migrationsDir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", "CREATE TABLE users (id int);").
			Build()

		// Act
		plan, err := GeneratePlan(
			t.Context(),
			fs,
			config,
			migrationsDir,
			filepath.Join(baseDir, "schema.sql"),
			nil,
		)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, plan.Statements)
	})
}

func TestDumpSchema(t *testing.T) {
	t.Parallel()

	const schema = `
CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE posts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users (id),
    title text NOT NULL
);
`

	t.Run("should dump DDL for all tables, when database has schema", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool := poolFactory.Pool(t)
		connConfig := pool.Config().ConnConfig.Copy()

		testutil.Exec(t, pool, schema)

		// Act
		stmts, err := DumpSchema(t.Context(), connConfig, nil)

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, stmts)
		snaps.MatchSnapshot(t, stmts)
	})

	t.Run("should return empty statements, when database has no user-defined objects", func(t *testing.T) {
		t.Parallel()

		// Arrange
		pool := poolFactory.Pool(t)
		connConfig := pool.Config().ConnConfig.Copy()

		// Act — DumpSchema on the base TEST_DATABASE_URL which has no user tables.
		stmts, err := DumpSchema(t.Context(), connConfig, nil)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stmts)
	})

	t.Run("should exclude conduit_migrations from dump, when database has conduit tables", func(t *testing.T) {
		t.Parallel()

		// Arrange — create a database with both user tables and conduit internal tables.
		pool := poolFactory.Pool(t)
		connConfig := pool.Config().ConnConfig.Copy()

		testutil.Exec(t, pool, schema)
		testutil.Exec(t, pool, string(migrations.Schema))

		// Act
		stmts, err := DumpSchema(t.Context(), connConfig, nil)

		// Assert
		require.NoError(t, err)

		for _, stmt := range stmts {
			assert.NotContains(t, stmt.DDL, "conduit_migrations",
				"dump should not include conduit internal tables")
		}
	})
}
