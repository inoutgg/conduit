package conduitcli

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/command/apply"
	"go.inout.gg/conduit/internal/command/cmdutil"
	"go.inout.gg/conduit/internal/command/create"
	"go.inout.gg/conduit/internal/command/initialise"
)

// Execute evaluates given os.Args and executes a matched command.
func Execute(ctx context.Context, migrator *conduit.Migrator) error {
	//nolint:exhaustruct
	cmd := &cli.Command{
		Name:  "conduit",
		Usage: "An SQL migrator that is easy to embed.",
		Flags: cmdutil.GlobalFlags,
		Commands: []*cli.Command{
			initialise.NewCommand(),
			create.NewCommand(),
			apply.NewCommand(migrator),
		},
		Before: cmdutil.OnBeforeHook,
	}

	//nolint:wrapcheck
	return cmd.Run(ctx, os.Args)
}
