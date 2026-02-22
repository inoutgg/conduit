package initialise

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/internal/timegenerator"
	"go.inout.gg/conduit/pkg/conduitsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/version"
)

//nolint:revive // ignore naming convention.
type InitialiseArgs struct {
	Dir         string
	DatabaseURL string
}

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    flagname.MigrationsDir,
				Usage:   "directory with migration files",
				Value:   "./migrations",
				Sources: cli.EnvVars("CONDUIT_MIGRATION_DIR"),
			},
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    flagname.DatabaseURL,
				Usage:   "database connection URL",
				Sources: cli.EnvVars("CONDUIT_DATABASE_URL"),
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := InitialiseArgs{
				Dir:         filepath.Clean(cmd.String(flagname.MigrationsDir)),
				DatabaseURL: cmd.String(flagname.DatabaseURL),
			}

			return initialise(ctx, fs, timeGen, args)
		},
	}
}

func initialise(ctx context.Context, fs afero.Fs, timeGen timegenerator.Generator, args InitialiseArgs) error {
	if err := createMigrationDir(fs, args.Dir); err != nil {
		return err
	}

	ver := version.NewFromTime(timeGen.Now())

	if err := createInitialMigration(fs, ver, args.Dir); err != nil {
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

	sumPath := filepath.Join(args.Dir, conduitsum.Filename)
	if err := afero.WriteFile(fs, sumPath, conduitsum.Format([]string{hash}), 0o644); err != nil {
		return fmt.Errorf("conduit: failed to write conduit.sum: %w", err)
	}

	return nil
}

// createMigrationDir creates a new migration directory.
func createMigrationDir(fs afero.Fs, dir string) error {
	err := fs.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("conduit: failed to create migrations directory at %s: %w", dir, err)
	}

	return nil
}

// createInitialMigration writes the initial conduit schema migration into the
// migrations directory.
func createInitialMigration(fs afero.Fs, ver version.Version, dir string) error {
	filename := version.MigrationFilename(ver, "conduit_initial_schema", version.MigrationDirectionUp)
	path := filepath.Join(dir, filename)

	if err := afero.WriteFile(fs, path, migrations.Schema, 0o644); err != nil {
		return fmt.Errorf("conduit: failed to create initial migration file: %w", err)
	}

	return nil
}
