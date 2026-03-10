package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/migrationfile"
	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/pkg/hashsum"
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

// Rehash recomputes the schema hash from existing migrations and persists it
// to conduit.sum.
func Rehash(
	ctx context.Context,
	fs afero.Fs,
	store hashsum.Store,
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

	stmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return fmt.Errorf("failed to parse conduit internal schema: %w", err)
	}

	migrationStmts, err := migrationfile.ReadStmtsFromDir(fs, args.MigrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	stmts = append(stmts, migrationStmts...)

	hash, err := pgdiff.GenerateSchemaHash(ctx, connConfig, stmts, args.ExcludeSchemas)
	if err != nil {
		return fmt.Errorf("failed to generate schema hash: %w", err)
	}

	if err := store.Save(args.RootDir, []byte(hash)); err != nil {
		return fmt.Errorf("failed to write conduit.sum: %w", err)
	}

	return nil
}
