package conduitcli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("should return error, when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		args := DiffArgs{
			Dir:         "/nonexistent",
			Name:        "add_posts",
			SchemaPath:  "/schema.sql",
			DatabaseURL: "postgres://localhost:5432/testdb",
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "migrations directory does not exist")
	})

	t.Run("should return error, when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  "/schema.sql",
			DatabaseURL: "://invalid",
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse database URL")
	})

	t.Run("should create migration file, when schema has new table", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, dir)
	})

	t.Run("should return error, when source schema hash does not match conduit.sum", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithFile("conduit.sum", "0000000000000000").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.Error(t, err)
		require.ErrorContains(t, err, "source schema drift detected")

		// No migration file should have been created.
		entries, _ := afero.ReadDir(fs, dir)
		for _, e := range entries {
			assert.NotContains(t, e.Name(), "add_posts",
				"migration file should not be created on drift")
		}
	})

	t.Run("should succeed, when source schema hash matches conduit.sum", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// First run to generate the correct conduit.sum.
		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}
		require.NoError(t, Diff(t.Context(), fs, timeGen, args))

		// Update schema to trigger a new diff, using the existing conduit.sum
		// which now contains the correct target hash from the first run.
		require.NoError(
			t,
			afero.WriteFile(fs, filepath.Join(baseDir, "schema.sql"), []byte(`CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);
CREATE TABLE comments (id int, post_id int);`), 0o644),
		)

		args2 := DiffArgs{
			Dir:         dir,
			Name:        "add_comments",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		// Act — second diff should succeed because the source hash matches conduit.sum.
		err := Diff(t.Context(), fs, timegenerator.Stub{
			T: time.Date(2024, 2, 15, 12, 30, 45, 0, time.UTC),
		}, args2)

		require.NoError(t, err)
	})

	t.Run("should include disable-tx directive, when diff contains concurrent index", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE INDEX idx_users_id ON users (id);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_index",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.NoError(t, err)

		migrationFile := filepath.Join(dir, "20240115123045_add_index.up.sql")
		content, err := afero.ReadFile(fs, migrationFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "---- disable-tx ----")
	})

	t.Run("should omit disable-tx directive, when diff has no concurrent DDL", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.NoError(t, err)

		migrationFile := filepath.Join(dir, "20240115123045_add_posts.up.sql")
		content, err := afero.ReadFile(fs, migrationFile)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "---- disable-tx ----")
	})

	t.Run("should return error, when no schema changes detected", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", "CREATE TABLE users (id int);").
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "no_changes",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "no schema changes detected")
	})
}

func TestSplitMigrations(t *testing.T) {
	t.Parallel()

	stmt := func(ddl string) schemadiff.Statement {
		return schemadiff.Statement{DDL: ddl}
	}

	t.Run("should return empty, when no statements provided", func(t *testing.T) {
		t.Parallel()

		migrations := splitMigrations(nil)
		assert.Empty(t, migrations)
	})

	t.Run("should produce single tx group, when all statements are transactional", func(t *testing.T) {
		t.Parallel()

		migrations := splitMigrations([]schemadiff.Statement{
			stmt("CREATE TABLE t1 (id int)"),
			stmt("CREATE TABLE t2 (id int)"),
		})
		require.Len(t, migrations, 1)
		assert.False(t, migrations[0].isNonTx)
		assert.Len(t, migrations[0].stmts, 2)
	})

	t.Run("should produce single non-tx group, when all statements are concurrent", func(t *testing.T) {
		t.Parallel()

		migrations := splitMigrations([]schemadiff.Statement{
			stmt("CREATE INDEX CONCURRENTLY idx1 ON t (c)"),
			stmt("CREATE INDEX CONCURRENTLY idx2 ON t (c)"),
		})
		require.Len(t, migrations, 1)
		assert.True(t, migrations[0].isNonTx)
		assert.Len(t, migrations[0].stmts, 2)
	})

	t.Run("should split into three groups, when statements alternate tx and non-tx", func(t *testing.T) {
		t.Parallel()

		migrations := splitMigrations([]schemadiff.Statement{
			stmt("CREATE TABLE t1 (id int)"),
			stmt("CREATE TABLE t2 (id int)"),
			stmt("CREATE INDEX CONCURRENTLY idx ON t1 (id)"),
			stmt("DROP INDEX CONCURRENTLY old_idx"),
			stmt("ALTER TABLE t1 ADD COLUMN name text"),
		})
		require.Len(t, migrations, 3)

		assert.False(t, migrations[0].isNonTx)
		assert.Len(t, migrations[0].stmts, 2)

		assert.True(t, migrations[1].isNonTx)
		assert.Len(t, migrations[1].stmts, 2)

		assert.False(t, migrations[2].isNonTx)
		assert.Len(t, migrations[2].stmts, 1)
	})

	t.Run("should produce single non-tx group, when one concurrent statement exists", func(t *testing.T) {
		t.Parallel()

		migrations := splitMigrations([]schemadiff.Statement{
			stmt("CREATE INDEX CONCURRENTLY idx ON t (c)"),
		})
		require.Len(t, migrations, 1)
		assert.True(t, migrations[0].isNonTx)
		assert.Len(t, migrations[0].stmts, 1)
	})
}

func TestDiffSplitsMigrations(t *testing.T) {
	t.Parallel()

	t.Run("should split into tx and non-tx files, when diff contains mixed DDL", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);
CREATE INDEX idx_posts_user_id ON posts (user_id);`).
			Build()

		args := DiffArgs{
			Dir:         dir,
			Name:        "add_posts",
			SchemaPath:  filepath.Join(baseDir, "schema.sql"),
			DatabaseURL: databaseURL,
		}

		err := Diff(t.Context(), fs, timeGen, args)

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, dir)
	})
}

func TestRequiresDisableTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ddl  string
		want bool
	}{
		{"CREATE INDEX CONCURRENTLY", "CREATE INDEX CONCURRENTLY idx ON t (c)", true},
		{"CREATE UNIQUE INDEX CONCURRENTLY", "CREATE UNIQUE INDEX CONCURRENTLY idx ON t (c)", true},
		{"DROP INDEX CONCURRENTLY", "DROP INDEX CONCURRENTLY idx", true},
		{"REINDEX INDEX CONCURRENTLY", "REINDEX INDEX CONCURRENTLY idx", true},
		{"REINDEX TABLE CONCURRENTLY", "REINDEX TABLE CONCURRENTLY t", true},
		{"ALTER TYPE ADD VALUE", "ALTER TYPE my_enum ADD VALUE 'new_val'", true},
		{"ALTER TYPE ADD VALUE with BEFORE", "ALTER TYPE my_enum ADD VALUE 'x' BEFORE 'y'", true},
		{"lowercase create index concurrently", "create index concurrently idx on t (c)", true},
		{"mixed case", "Create Index Concurrently idx ON t (c)", true},
		{"CREATE INDEX without CONCURRENTLY", "CREATE INDEX idx ON t (c)", false},
		{"DROP INDEX without CONCURRENTLY", "DROP INDEX idx", false},
		{"CREATE TABLE", "CREATE TABLE t (id int)", false},
		{"ALTER TABLE ADD COLUMN", "ALTER TABLE t ADD COLUMN c int", false},
		{"ALTER TYPE RENAME", "ALTER TYPE my_enum RENAME TO other_enum", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isNonTxStmt(tt.ddl))
		})
	}
}
