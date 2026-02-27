package create

import (
	"context"
	"errors"
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
		Name:  "create",
		Usage: "create a new migration file",
		Commands: []*cli.Command{
			//nolint:exhaustruct
			{
				Name:  "diff",
				Usage: "create a migration from schema diff using pg-schema-diff",
				Flags: []cli.Flag{
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:     "schema",
						Usage:    "path to the target schema SQL file",
						Required: true,
					},
					commandutil.DatabaseURLFlag(false),
					commandutil.MigrationsDirFlag(),
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().First()
					if name == "" {
						return errors.New("missing `name` argument")
					}

					args := conduitcli.DiffArgs{
						Dir:         filepath.Clean(cmd.String(commandutil.MigrationsDir)),
						Name:        name,
						SchemaPath:  cmd.String("schema"),
						DatabaseURL: cmd.String(commandutil.DatabaseURL),
					}

					return conduitcli.Diff(ctx, fs, timeGen, args)
				},
			},
		},
	}
}
