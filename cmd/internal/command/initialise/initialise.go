package initialise

import (
	"context"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/flagname"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/pkg/timegenerator"
)

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
			args := conduitcli.InitArgs{
				Dir:         filepath.Clean(cmd.String(flagname.MigrationsDir)),
				DatabaseURL: cmd.String(flagname.DatabaseURL),
			}

			return conduitcli.Init(ctx, fs, timeGen, args)
		},
	}
}
