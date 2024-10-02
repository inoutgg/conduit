package conduitcli

import (
	"context"
	"os"

	"github.com/urfave/cli/v2"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/command/apply"
	"go.inout.gg/conduit/internal/command/create"
	"go.inout.gg/conduit/internal/command/initialise"
	"go.inout.gg/conduit/internal/command/root"
)

// Execute evaluates given os.Args and executes a matched command.
func Execute(ctx context.Context, migrator conduit.Migrator) error {
	cmd := &cli.App{
		Flags: root.GlobalFlags,
		Commands: []*cli.Command{
			initialise.NewCommand(),
			create.NewCommand(),
			apply.NewCommand(migrator),
		},
		Before: root.OnBeforeHook,
	}

	return cmd.Run(os.Args)
}
