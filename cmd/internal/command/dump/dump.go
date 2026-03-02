package dump

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/commandutil"
	"go.inout.gg/conduit/cmd/internal/config"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/pkg/buildinfo"
)

func NewCommand(bi buildinfo.BuildInfo, cfg *config.Config) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "dump",
		Usage: "dump schema DDL from a remote Postgres database",
		Flags: []cli.Flag{
			commandutil.DatabaseURLFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			url := commandutil.StringOr(cmd, commandutil.DatabaseURL, cfg.Database.URL)
			if url == "" {
				return fmt.Errorf("missing `%s' flag", commandutil.DatabaseURL)
			}

			return conduitcli.Dump(ctx, os.Stdout, bi, conduitcli.DumpArgs{DatabaseURL: url})
		},
	}
}
