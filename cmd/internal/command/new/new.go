//nolint:predeclared
package new

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/timegenerator"
)

//nolint:revive
func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	timeGen timegenerator.Generator,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:      "new",
		Usage:     "create a new empty migration",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			cmdutil.MigrationsDirFlag(src),
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return errors.New("missing required argument: <name>")
			}

			result, err := conduitcli.New(fs, timeGen, conduitcli.NewArgs{
				MigrationsDir: cmd.String(cmdutil.MigrationsDir),
				Name:          name,
			})
			if err != nil {
				return fmt.Errorf("failed to create migration: %w", err)
			}

			fmt.Fprintf(stderr, "Created %s\n", result.UpFile)
			fmt.Fprintf(stderr, "Created %s\n", result.DownFile)

			return nil
		},
	}
}
