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

func exec(t *testing.T, fs afero.Fs, stdout *bytes.Buffer, args string) error {
	t.Helper()

	//nolint:wrapcheck
	return command.Execute(t.Context(), fs, stdout, timeGen, bi, "/", strings.Fields(args))
}

func bootstrap(t *testing.T, fs afero.Fs, dbURL string) {
	t.Helper()

	var stdout bytes.Buffer

	require.NoError(t, exec(t, fs, &stdout, "conduit init --database-url "+dbURL))
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

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit init --database-url "+testutil.ConnString(pool))

		require.NoError(t, err)
		snapshotConfig(t, fs)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, stdout.String())
	})

	t.Run("should initialise migration directory, when custom migrations dir is specified", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit init --database-url "+
			testutil.ConnString(pool)+" --migrations-dir ./custom")

		require.NoError(t, err)
		snapshotConfig(t, fs)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, stdout.String())
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

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit diff --database-url "+dbURL+" --schema schema.sql add_posts")

		require.NoError(t, err)
		testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
		snaps.MatchSnapshot(t, stdout.String())
	})

	t.Run("should return error, when name argument is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit diff --database-url "+
			testutil.ConnString(pool)+" --schema schema.sql")

		require.ErrorContains(t, err, "missing required argument: <name>")
	})

	t.Run("should return error, when schema flag is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit diff --database-url "+testutil.ConnString(pool)+" add_posts")

		require.ErrorContains(t, err, "missing required flag: --schema")
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

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit apply --database-url "+dbURL+" up")

		require.NoError(t, err)
		assert.True(t, testutil.TableExists(t, pool, "conduit_migrations"))
		snaps.MatchSnapshot(t, stdout.String())
	})

	t.Run("should return error, when schema drift is detected", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		// Apply initial migration to store a schema hash.
		var stdout bytes.Buffer
		require.NoError(t, exec(t, fs, &stdout, "conduit apply --database-url "+dbURL+" up"))

		// Manually alter the schema outside of migrations.
		testutil.Exec(t, pool, "CREATE TABLE rogue_table (id int)")

		// Apply again — drift check should detect the mismatch.
		stdout.Reset()
		err := exec(t, fs, &stdout, "conduit apply --database-url "+dbURL+" up")

		require.ErrorContains(t, err, "schema drift detected")
	})

	t.Run("should return error, when database-url is missing", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit apply up")

		require.ErrorContains(t, err, "missing required flag: --database-url")
	})

	t.Run("should return error, when direction is missing", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit apply --database-url "+testutil.ConnString(pool))

		require.ErrorContains(t, err, "failed to parse direction")
	})

	t.Run("should preview migrations, when dry-run is enabled", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		fs := afero.NewMemMapFs()
		dbURL := testutil.ConnString(pool)

		bootstrap(t, fs, dbURL)

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit apply --database-url "+dbURL+" --dry-run up")

		require.NoError(t, err)
		snaps.MatchSnapshot(t, stdout.String())
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

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit dump --database-url "+testutil.ConnString(pool))

		require.NoError(t, err)
		snaps.MatchSnapshot(t, stdout.String())
	})

	t.Run("should return error, when database-url is missing", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()

		var stdout bytes.Buffer

		err := exec(t, fs, &stdout, "conduit dump")

		require.ErrorContains(t, err, "missing required flag: --database-url")
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
	var diffStdout bytes.Buffer

	err := exec(t, fs, &diffStdout, "conduit diff --database-url "+dbURL+" --schema schema.sql add_tables")
	require.NoError(t, err)

	// 4. Apply: run all migrations (init + generated).
	var applyStdout bytes.Buffer

	err = exec(t, fs, &applyStdout, "conduit apply --database-url "+dbURL+" up")
	require.NoError(t, err)

	assert.True(t, testutil.TableExists(t, pool, "conduit_migrations"))
	assert.True(t, testutil.TableExists(t, pool, "users"))
	assert.True(t, testutil.TableExists(t, pool, "posts"))
	testutil.SnapshotFS(t, fs, ".", "conduit.yaml")
	snaps.MatchSnapshot(t, diffStdout.String(), applyStdout.String())
}
