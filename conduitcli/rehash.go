package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/migrationfile"
	"go.inout.gg/conduit/pkg/lockfile"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
)

// RehashArgs configures a [Rehash] operation.
type RehashArgs struct {
	RootDir        string
	MigrationsDir  string
	DatabaseURL    string
	ExcludeSchemas []string
}

// Rehash recomputes the schema hash chain from existing migrations and
// persists it to conduit.lock.
func Rehash(
	ctx context.Context,
	fs afero.Fs,
	store lockfile.Store,
	args RehashArgs,
) error {
	if !exists(fs, args.MigrationsDir) {
		return fmt.Errorf("%w: directory %q does not exist",
			ErrMigrationsNotFound, args.MigrationsDir)
	}

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
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

	entries := make([]lockfile.Entry, len(migrations))
	for i, m := range migrations {
		entries[i] = lockfile.Entry{Parsed: m.Parsed, Hash: hashes[i]}
	}

	if err := store.Save(args.RootDir, entries); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}
