package conduitcli

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
	"go.inout.gg/conduit/pkg/hashsum"
)

func TestRehash(t *testing.T) {
	t.Parallel()

	t.Run("should return error, when migrations directory does not exist", func(t *testing.T) {
		t.Parallel()

		fs := afero.NewMemMapFs()
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := RehashArgs{
			RootDir:       "/",
			MigrationsDir: "/nonexistent",
			DatabaseURL:   "postgres://localhost:5432/testdb",
		}

		err := Rehash(t.Context(), fs, store, args)

		require.Error(t, err)
		require.ErrorIs(t, err, ErrMigrationsNotFound)
	})

	t.Run("should return error, when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		fs, _, dir := testutil.NewMigrationsDirBuilder(t).Build()
		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := RehashArgs{
			RootDir:       "/",
			MigrationsDir: dir,
			DatabaseURL:   "://invalid",
		}

		err := Rehash(t.Context(), fs, store, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse database URL")
	})

	t.Run("should update conduit.sum with correct hash", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("conduit.sum", "0000000000000000").
			Build()

		store := hashsum.NewFSStore(fs, "conduit.sum")
		args := RehashArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			DatabaseURL:   databaseURL,
		}

		err := Rehash(t.Context(), fs, store, args)

		require.NoError(t, err)

		// Verify the hash was updated from the stale value.
		sum, err := afero.ReadFile(fs, baseDir+"/conduit.sum")
		require.NoError(t, err)
		assert.NotEqual(t, "0000000000000000", string(sum))
		assert.NotEmpty(t, string(sum))
	})

	t.Run("should produce hash consistent with diff", func(t *testing.T) {
		t.Parallel()

		databaseURL := os.Getenv("TEST_DATABASE_URL")
		fs, baseDir, dir := testutil.NewMigrationsDirBuilder(t).
			WithFile("20230601120000_init.up.sql", "CREATE TABLE users (id int);").
			WithBaseFile("schema.sql", `CREATE TABLE users (id int);
CREATE TABLE posts (id int, user_id int);`).
			Build()

		// Run diff to generate a migration and conduit.sum.
		store := hashsum.NewFSStore(fs, "conduit.sum")
		_, err := Diff(t.Context(), fs, timeGen, bi, store, DiffArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			Name:          "add_posts",
			SchemaPath:    baseDir + "/schema.sql",
			DatabaseURL:   databaseURL,
		})
		require.NoError(t, err)

		diffSum, err := afero.ReadFile(fs, baseDir+"/conduit.sum")
		require.NoError(t, err)

		// Corrupt the sum file.
		require.NoError(t, afero.WriteFile(fs, baseDir+"/conduit.sum", []byte("corrupted"), 0o644))

		// Rehash should restore the correct hash.
		err = Rehash(t.Context(), fs, store, RehashArgs{
			RootDir:       baseDir,
			MigrationsDir: dir,
			DatabaseURL:   databaseURL,
		})
		require.NoError(t, err)

		rehashSum, err := afero.ReadFile(fs, baseDir+"/conduit.sum")
		require.NoError(t, err)
		assert.Equal(t, string(diffSum), string(rehashSum))
	})
}
