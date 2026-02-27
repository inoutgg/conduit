package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/pkg/conduitsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/timegenerator"
	"go.inout.gg/conduit/pkg/version"
)

type InitArgs struct {
	Dir         string
	DatabaseURL string
}

func Init(ctx context.Context, fs afero.Fs, timeGen timegenerator.Generator, args InitArgs) error {
	if err := createMigrationDir(fs, args.Dir); err != nil {
		return err
	}

	migrationsFs := afero.NewBasePathFs(fs, args.Dir)
	ver := version.NewFromTime(timeGen.Now())

	if err := createInitialMigration(migrationsFs, ver); err != nil {
		return err
	}

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	stmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return fmt.Errorf("failed to parse initial schema: %w", err)
	}

	hash, err := pgdiff.GenerateSchemaHash(ctx, connConfig, stmts)
	if err != nil {
		return fmt.Errorf("failed to generate schema hash: %w", err)
	}

	if err := conduitsum.WriteFile(migrationsFs, hash); err != nil {
		return fmt.Errorf("conduit: failed to write conduit.sum: %w", err)
	}

	return nil
}

func createMigrationDir(fs afero.Fs, dir string) error {
	err := fs.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("conduit: failed to create migrations directory at %s: %w", dir, err)
	}

	return nil
}

func createInitialMigration(fs afero.Fs, ver version.Version) error {
	filename := version.MigrationFilename(ver, "conduit_initial_schema", version.MigrationDirectionUp)

	if err := afero.WriteFile(fs, filename, migrations.Schema, 0o644); err != nil {
		return fmt.Errorf("conduit: failed to create initial migration file: %w", err)
	}

	return nil
}
