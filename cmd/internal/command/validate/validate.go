package validate

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
	"go.inout.gg/conduit/pkg/lockfile"
)

func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "validate",
		Usage: "validate the migration chain against conduit.lock",
		Flags: []cli.Flag{
			cmdutil.ExcludeSchemasFlag(src),
			cmdutil.DatabaseURLFlag(src),
			cmdutil.MigrationsDirFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store := lockfile.NewFSStore(fs, "conduit.lock")
			args := conduitcli.ValidateArgs{
				RootDir:        ".",
				MigrationsDir:  filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			if err := conduitcli.Validate(ctx, fs, store, args); err != nil {
				return fmt.Errorf("failed to validate: %w", err)
			}

			fmt.Fprintln(stderr, "Migration chain is valid.")

			return nil
		},
	}
}
