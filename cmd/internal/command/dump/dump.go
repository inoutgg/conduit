package dump

import (
	"context"
	"fmt"
	"os"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/buildinfo"
)

func NewCommand(bi buildinfo.BuildInfo, src altsrc.Sourcer) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "dump",
		Usage: "dump schema DDL from a remote Postgres database",
		Flags: []cli.Flag{
			cmdutil.DatabaseURLFlag(src),
			cmdutil.ExcludeSchemasFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dbURL := cmd.String(cmdutil.DatabaseURL)
			if dbURL == "" {
				return fmt.Errorf("missing required flag: --%s", cmdutil.DatabaseURL)
			}

			return conduitcli.Dump(ctx, os.Stdout, bi, conduitcli.DumpArgs{
				DatabaseURL:    dbURL,
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			})
		},
	}
}
