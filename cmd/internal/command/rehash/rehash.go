package rehash

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
)

func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "rehash",
		Usage: "recompute conduit.sum from existing migrations",
		Flags: []cli.Flag{
			cmdutil.ExcludeSchemasFlag(src),
			cmdutil.DatabaseURLFlag(src),
			cmdutil.MigrationsDirFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store := hashsum.NewFSStore(fs, "conduit.sum")
			args := conduitcli.RehashArgs{
				RootDir:        ".",
				MigrationsDir:  filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			if err := conduitcli.Rehash(ctx, fs, store, args); err != nil {
				return fmt.Errorf("failed to rehash: %w", err)
			}

			fmt.Fprintln(stderr, "Updated conduit.sum")

			return nil
		},
	}
}
