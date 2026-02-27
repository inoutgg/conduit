package initialise

import (
	"context"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/commandutil"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			commandutil.MigrationsDirFlag(),
			commandutil.DatabaseURLFlag(false),
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := conduitcli.InitArgs{
				Dir:         filepath.Clean(cmd.String(commandutil.MigrationsDir)),
				DatabaseURL: cmd.String(commandutil.DatabaseURL),
			}

			return conduitcli.Init(ctx, fs, timeGen, args)
		},
	}
}
