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

var _ Interface = (*app)(nil)

// Inferface exposes public CLI interface.
type Interface interface {
	// Execute executes a command if matched.
	Execute(context.Context) error
}

type app struct {
	cmd *cli.App
}

// New creates a new command-line interface with a given dialer
// to connect to the database.
func New(m conduit.Migrator) Interface {
	cmd := &cli.App{
		Flags: root.Flags,
		Commands: []*cli.Command{
			initialise.NewCommand(),
			create.NewCommand(),
			apply.NewCommand(m),
		},
		Before: root.OnBeforeHook,
	}

	return &app{cmd}
}

func (c *app) Execute(ctx context.Context) error {
	return c.cmd.Run(os.Args)
}
