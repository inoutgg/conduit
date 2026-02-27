//nolint:wrapcheck
package apply

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/cmd/internal/command/commandutil"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/direction"
)

const (
	stepsFlag         = "steps"
	allowHazardsFlag  = "allow-hazards"
	noSchemaDriftFlag = "no-check-schema-drift"
)

func NewCommand(fs afero.Fs) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "apply",
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			commandutil.DatabaseURLFlag(true),
			commandutil.MigrationsDirFlag(),

			//nolint:exhaustruct
			&cli.IntFlag{
				Name:  stepsFlag,
				Usage: "maximum migrations steps",
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  allowHazardsFlag,
				Usage: "allow applying migrations that contain hazardous operations",
				Value: false,
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  noSchemaDriftFlag,
				Usage: "skip check for schema drift before applying migrations",
				Value: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, err := direction.FromString(cmd.Args().First())
			if err != nil {
				return fmt.Errorf("failed to parse direction: %w", err)
			}

			url := cmd.String(commandutil.DatabaseURL)
			if url == "" {
				return fmt.Errorf("missing `%s' flag", commandutil.DatabaseURL)
			}

			migrationsDir := cmd.String(commandutil.MigrationsDir)
			registry := conduitregistry.FromFS(fs, migrationsDir)
			migrator := conduit.NewMigrator(conduit.WithRegistry(registry))

			args := conduitcli.ApplyArgs{
				DatabaseURL:          url,
				Direction:            dir,
				Steps:                cmd.Int(stepsFlag),
				AllowHazards:         cmd.Bool(allowHazardsFlag),
				SkipSchemaDriftCheck: cmd.Bool(noSchemaDriftFlag),
			}

			err = conduitcli.Apply(ctx, migrator, args)
			if err != nil && errors.Is(err, conduit.ErrHazardDetected) {
				return fmt.Errorf("%w\n\nuse --%s to proceed", err, allowHazardsFlag)
			}

			return err
		},
	}
}
