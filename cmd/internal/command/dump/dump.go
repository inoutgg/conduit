package dump

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/flagname"
	"go.inout.gg/conduit/conduitcli"
)

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "dump",
		Usage: "dump schema DDL from a remote Postgres database",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:     flagname.DatabaseURL,
				Usage:    "database connection URL",
				Sources:  cli.EnvVars("CONDUIT_DATABASE_URL"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := conduitcli.DumpArgs{
				DatabaseURL: cmd.String(flagname.DatabaseURL),
			}

			return conduitcli.Dump(ctx, args, os.Stdout)
		},
	}
}
