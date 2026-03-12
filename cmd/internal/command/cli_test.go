package command_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/cmd/internal/command"
	"go.inout.gg/conduit/internal/testutil"
)

type execResult struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func exec(t *testing.T, fs afero.Fs, args string) (execResult, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	err := command.Execute(t.Context(), fs, &stdout, &stderr, timeGen, bi, sw, "/", strings.Fields(args))

	return execResult{stdout: &stdout, stderr: &stderr}, err //nolint:wrapcheck
}

func bootstrap(t *testing.T, fs afero.Fs, dbURL string) {
	t.Helper()

	_, err := exec(t, fs, "conduit init --database-url "+dbURL)
	require.NoError(t, err)
}

func snapshotConfig(t *testing.T, fs afero.Fs) {
	t.Helper()

	b, err := afero.ReadFile(fs, "conduit.yaml")
	require.NoError(t, err)

	snaps.MatchYAML(t, string(b), match.Any("$.database.url"))
}

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("should initialise migration directory, when using default settings", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		r, err := exec(t, fs, "conduit init --database-url "+testutil.ConnString(pool))

		require.NoError(t, err)
		snapshotConfig(t, fs)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, r.stderr.String())
	})

	t.Run("should initialise migration directory, when custom migrations dir is specified", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		r, err := exec(t, fs, "conduit init --database-url "+
			testutil.ConnString(pool)+" --migrations-dir ./custom")

		require.NoError(t, err)
		snapshotConfig(t, fs)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, r.stderr.String())
	})
}

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("should create migration file, when schema diff exists", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)
		require.NoError(t, afero.WriteFile(
			fs, "schema.sql",
			[]byte("CREATE TABLE posts (id int, user_id int);"), 0o644,
		))

		r, err := exec(t, fs, "conduit diff --database-url "+dbURL+" --schema schema.sql add_posts")

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, r.stderr.String())
	})

	t.Run("should return error, when name argument is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		_, err := exec(t, fs, "conduit diff --database-url "+
			testutil.ConnString(pool)+" --schema schema.sql")

		require.ErrorContains(t, err, "missing required argument: <name>")
	})

	t.Run("should return error, when schema flag is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		_, err := exec(t, fs, "conduit diff --database-url "+testutil.ConnString(pool)+" add_posts")

		require.ErrorContains(t, err, "Required flag \"schema\" not set")
	})

	t.Run("should report no changes, when schema is already in sync", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		// Write a target schema that matches the initial migration exactly.
		require.NoError(t, afero.WriteFile(
			fs, "schema.sql",
			[]byte(`CREATE TABLE IF NOT EXISTS conduit_migrations (
  id BIGSERIAL NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  version VARCHAR(255) NOT NULL,
  name VARCHAR(4095) NOT NULL,
  hash VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE (version, name)
);`), 0o644,
		))

		r, err := exec(t, fs, "conduit diff --database-url "+dbURL+" --schema schema.sql no_op")

		require.NoError(t, err)
		snaps.MatchSnapshot(t, r.stderr.String())
	})
}

func TestApply(t *testing.T) {
	t.Parallel()

	t.Run("should apply migrations up", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		r, err := exec(t, fs, "conduit apply --database-url "+dbURL+" up")

		require.NoError(t, err)
		assert.True(t, testutil.TableExists(t, pool, "conduit_migrations"))
		snaps.MatchSnapshot(t, r.stderr.String())
	})

	t.Run("should return error, when schema drift is detected", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		// Apply initial migration to store a schema hash.
		_, err := exec(t, fs, "conduit apply --database-url "+dbURL+" up")
		require.NoError(t, err)

		// Manually alter the schema outside of migrations.
		testutil.Exec(t, pool, "CREATE TABLE rogue_table (id int)")

		// Apply again — drift check should detect the mismatch.
		_, err = exec(t, fs, "conduit apply --database-url "+dbURL+" up")

		require.ErrorContains(t, err, "schema drift detected")
	})

	t.Run("should return error, when database-url is missing", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()

		_, err := exec(t, fs, "conduit apply up")

		require.ErrorContains(t, err, "Required flag \"database-url\" not set")
	})

	t.Run("should return error, when direction is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		_, err := exec(t, fs, "conduit apply --database-url "+testutil.ConnString(pool))

		require.ErrorContains(t, err, "failed to parse direction")
	})

	t.Run("should preview migrations, when dry-run is enabled", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		r, err := exec(t, fs, "conduit apply --database-url "+dbURL+" --dry-run up")

		require.NoError(t, err)
		snaps.MatchSnapshot(t, r.stdout.String(), r.stderr.String())
	})
}

func TestDump(t *testing.T) {
	t.Parallel()

	t.Run("should dump schema DDL", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		testutil.Exec(t, pool, `
CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE posts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users (id),
    title text NOT NULL
);
`)

		fs := afero.NewMemMapFs()

		r, err := exec(t, fs, "conduit dump --database-url "+testutil.ConnString(pool))

		require.NoError(t, err)
		snaps.MatchSnapshot(t, r.stdout.String())
	})

	t.Run("should return error, when database-url is missing", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()

		_, err := exec(t, fs, "conduit dump")

		require.ErrorContains(t, err, "Required flag \"database-url\" not set")
	})
}

func TestInitDiffApply(t *testing.T) {
	t.Parallel()

	pool := poolFactory.Pool(t)
	fs := afero.NewMemMapFs()
	dbURL := testutil.ConnString(pool)

	// 1. Init: bootstrap the project.
	bootstrap(t, fs, dbURL)

	// 2. Write a target schema with user tables.
	require.NoError(t, afero.WriteFile(
		fs, "schema.sql",
		[]byte("CREATE TABLE users (id int);\nCREATE TABLE posts (id int, user_id int);"), 0o644,
	))

	// 3. Diff: generate migration from schema diff.
	diffResult, err := exec(t, fs, "conduit diff --database-url "+dbURL+" --schema schema.sql add_tables")
	require.NoError(t, err)

	// 4. Apply: run all migrations (init + generated).
	applyResult, err := exec(t, fs, "conduit apply --database-url "+dbURL+" up")
	require.NoError(t, err)

	assert.True(t, testutil.TableExists(t, pool, "conduit_migrations"))
	assert.True(t, testutil.TableExists(t, pool, "users"))
	assert.True(t, testutil.TableExists(t, pool, "posts"))
	testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
	snaps.MatchSnapshot(t, diffResult.stderr.String(), applyResult.stderr.String())
}
