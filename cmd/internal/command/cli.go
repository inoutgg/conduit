package command

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/apply"
	"go.inout.gg/conduit/cmd/internal/command/commandutil"
	"go.inout.gg/conduit/cmd/internal/command/create"
	"go.inout.gg/conduit/cmd/internal/command/dump"
	"go.inout.gg/conduit/cmd/internal/command/initialise"
	"go.inout.gg/conduit/cmd/internal/config"
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/timegenerator"
)

// Execute evaluates given os.Args and executes a matched command.
func Execute(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	fs := afero.NewBasePathFs(afero.NewOsFs(), cwd)

	var (
		timeGen timegenerator.Standard
		bi      buildinfo.Standard
		cfg     config.Config
	)

	//nolint:exhaustruct
	cmd := &cli.Command{
		Name:  "conduit",
		Usage: "An SQL migrator that is easy to embed.",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			var err error

			cfg, err = config.FromFS(afero.NewOsFs(), cmd.String("config"))
			if err != nil {
				return ctx, fmt.Errorf("failed to load config: %w", err)
			}

			return ctx, nil
		},
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to config file (default: conduit.yaml or .conduit.yaml)",
				Sources: cli.EnvVars("CONDUIT_CONFIG"),
			},
			commandutil.VerboseFlag("verbose mode"),
		},
		Commands: []*cli.Command{
			initialise.NewCommand(fs, timeGen, &cfg),
			create.NewCommand(fs, timeGen, bi, &cfg),
			apply.NewCommand(fs, &cfg),
			dump.NewCommand(bi, &cfg),
		},
	}

	//nolint:wrapcheck
	return cmd.Run(ctx, os.Args)
}
