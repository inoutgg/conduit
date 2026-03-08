package initialise

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	timeGen timegenerator.Generator,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			cmdutil.MigrationsDirFlag(src),
			cmdutil.DatabaseURLFlag(src),
			cmdutil.ExcludeSchemasFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store := hashsum.NewFSStore(fs, "conduit.sum")
			args := conduitcli.InitArgs{
				RootDir:        ".",
				ConfigName:     "conduit.yaml",
				MigrationsDir:  filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			result, err := conduitcli.Init(ctx, fs, timeGen, store, args)
			if err != nil {
				return fmt.Errorf("failed to initialise: %w", err)
			}

			fmt.Fprintln(stderr, "Created "+result.MigrationsDirPath)
			fmt.Fprintln(stderr, "Created "+result.MigrationPath)
			fmt.Fprintln(stderr, "Created "+result.ConfigPath)
			fmt.Fprintln(stderr, "Created "+result.SumPath)
			fmt.Fprintln(stderr)
			fmt.Fprintln(stderr, "Initialised conduit in "+result.MigrationsDirPath)

			return nil
		},
	}
}
