package initialise

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	timeGen timegenerator.Generator,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  cmdutil.MigrationsDir,
				Usage: "directory with migration files",
				Value: "./migrations",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_MIGRATIONS_DIR"),
				),
			},
			&cli.StringFlag{
				Name:     cmdutil.DatabaseURL,
				Usage:    "database connection URL",
				Required: true,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_DATABASE_URL"),
				),
			},
			&cli.StringSliceFlag{
				Name:  cmdutil.ExcludeSchemas,
				Usage: "PostgreSQL schemas to exclude",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_EXCLUDE_SCHEMAS"),
				),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store := hashsum.NewFSStore(fs, "conduit.sum")
			args := conduitcli.InitArgs{
				RootDir:        ".",
				ConfigName:     "conduit.yaml",
				MigrationsDir:  filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			result, err := conduitcli.Init(ctx, fs, timeGen, store, args)
			if err != nil {
				return fmt.Errorf("failed to initialise: %w", err)
			}

			fmt.Fprintln(stderr, "Created "+result.MigrationsDirPath)
			fmt.Fprintln(stderr, "Created "+result.MigrationPath)
			fmt.Fprintln(stderr, "Created "+result.ConfigPath)
			fmt.Fprintln(stderr, "Created "+result.SumPath)
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, "Initialised conduit in "+result.MigrationsDirPath)

			return nil
		},
	}
}
