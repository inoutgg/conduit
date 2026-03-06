package initialise

import (
	"context"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator, configSrc altsrc.Sourcer) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			cmdutil.MigrationsDirFlag(configSrc),
			cmdutil.DatabaseURLFlag(configSrc),
			cmdutil.ExcludeSchemasFlag(configSrc),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store := hashsum.NewFSStore(fs, "conduit.sum")
			args := conduitcli.InitArgs{
				RootDir:        ".",
				MigrationsDir:  filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			return conduitcli.Init(ctx, fs, timeGen, store, args)
		},
	}
}
