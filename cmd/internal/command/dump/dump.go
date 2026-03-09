package dump

import (
	"context"
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
			// we don't use alternative sources for this flag, since frankly
			// the database URL provided to this command is different from the
			// one used by other commands. Since typically this command target
			// against production database to derive the state.
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:     cmdutil.DatabaseURL,
				Usage:    "database connection URL",
				Required: true,
			},
			cmdutil.ExcludeSchemasFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return conduitcli.Dump(ctx, w, bi, conduitcli.DumpArgs{
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			})
		},
	}
}
