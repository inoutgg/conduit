package initialise

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator, src altsrc.Sourcer) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			cmdutil.MigrationsDirFlag(src),
			cmdutil.DatabaseURLFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := conduitcli.InitArgs{
				Dir:         filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL: cmd.String(cmdutil.DatabaseURL),
			}

			if err := conduitcli.Init(ctx, fs, timeGen, args); err != nil {
				return fmt.Errorf("conduit: init: %w", err)
			}

			return nil
		},
	}
}
