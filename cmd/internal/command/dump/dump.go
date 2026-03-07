package dump

import (
	"context"
	"fmt"
	"io"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
)

func NewCommand(w io.Writer, bi conduitbuildinfo.BuildInfo, src altsrc.Sourcer) *cli.Command {
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

			return conduitcli.Dump(ctx, w, bi, conduitcli.DumpArgs{
				DatabaseURL:    dbURL,
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			})
		},
	}
}
