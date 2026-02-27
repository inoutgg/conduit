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
	"go.inout.gg/conduit/pkg/timegenerator"
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
			&cli.BoolFlag{
				Name:  commandutil.Verbose,
				Usage: "verbose mode",
				Value: false,
			},
		},
		Commands: []*cli.Command{
			initialise.NewCommand(fs, timeGen),
			create.NewCommand(fs, timeGen),
			apply.NewCommand(fs),
			dump.NewCommand(),
		},
	}

	//nolint:wrapcheck
	return cmd.Run(ctx, os.Args)
}
