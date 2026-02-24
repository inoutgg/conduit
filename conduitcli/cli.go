package conduitcli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/apply"
	"go.inout.gg/conduit/internal/command/create"
	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/command/initialise"
	"go.inout.gg/conduit/internal/timegenerator"
)

// Execute evaluates given os.Args and executes a matched command.
func Execute(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), cwd)

	var timeGen timegenerator.Standard

	//nolint:exhaustruct
	cmd := &cli.Command{
		Name:  "conduit",
		Usage: "An SQL migrator that is easy to embed.",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    flagname.MigrationsDir,
				Usage:   "directory with migration files",
				Value:   "./migrations",
				Sources: cli.EnvVars("CONDUIT_MIGRATION_DIR"),
			},
			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  flagname.Verbose,
				Usage: "verbose mode",
				Value: false,
			},
		},
		Commands: []*cli.Command{
			initialise.NewCommand(fs, timeGen),
			create.NewCommand(fs, timeGen),
			apply.NewCommand(),
		},
	}

	//nolint:wrapcheck
	return cmd.Run(ctx, os.Args)
}
