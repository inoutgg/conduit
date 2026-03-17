package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/migrationfile"
	"go.inout.gg/conduit/pkg/lockfile"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
)

// ValidateArgs configures a [Validate] operation.
type ValidateArgs struct {
	RootDir        string
	MigrationsDir  string
	DatabaseURL    string
	ExcludeSchemas []string
}

// Validate checks that every migration in the lockfile still produces the
// recorded schema hash. It reports the first migration whose hash diverges.
func Validate(
	ctx context.Context,
	fs afero.Fs,
	store lockfile.Store,
	args ValidateArgs,
) error {
	if !exists(fs, args.MigrationsDir) {
		return fmt.Errorf("%w: directory %q does not exist",
			ErrMigrationsNotFound, args.MigrationsDir)
	}

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	entries, err := store.Read(args.RootDir)
	if err != nil {
		return fmt.Errorf("failed to read lockfile: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("lockfile is empty or does not exist")
	}

	migrations, err := migrationfile.ReadMigrationsFromDir(fs, args.MigrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	groups := make([][]sqlsplit.Stmt, len(migrations))
	for i, m := range migrations {
		groups[i] = m.Stmts
	}

	hashes, err := pgdiff.GenerateSchemaHashChain(ctx, connConfig, groups, args.ExcludeSchemas)
	if err != nil {
		return fmt.Errorf("failed to compute schema hash chain: %w", err)
	}

	return compareChain(entries, migrations, hashes)
}

// compareChain checks lockfile entries against computed hashes and reports
// the first mismatch.
func compareChain(entries []lockfile.Entry, migrations []migrationfile.Migration, hashes []string) error {
	if len(entries) != len(migrations) {
		return fmt.Errorf(
			"%w: lockfile has %d entries but found %d migrations",
			conduit.ErrSchemaDrift, len(entries), len(migrations),
		)
	}

	for i, entry := range entries {
		if entry.Parsed.Compare(migrations[i].Parsed) != 0 {
			return fmt.Errorf(
				"%w: lockfile entry %d expected migration %s, found %s",
				conduit.ErrSchemaDrift, i+1, entry.Parsed.String(), migrations[i].Parsed.String(),
			)
		}

		if entry.Hash != hashes[i] {
			return fmt.Errorf(
				"%w: migration %s has been modified (expected hash %s, got %s)",
				conduit.ErrSchemaDrift, entry.Parsed.String(), entry.Hash, hashes[i],
			)
		}
	}

	return nil
}
