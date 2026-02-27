package dump

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/commandutil"
	"go.inout.gg/conduit/conduitcli"
)

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "dump",
		Usage: "dump schema DDL from a remote Postgres database",
		Flags: []cli.Flag{
			commandutil.DatabaseURLFlag(true),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := conduitcli.DumpArgs{
				DatabaseURL: cmd.String(commandutil.DatabaseURL),
			}

			return conduitcli.Dump(ctx, args, os.Stdout)
		},
	}
}
