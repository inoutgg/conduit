//nolint:wrapcheck
package apply

import (
	"context"
	"errors"
	"fmt"
	"os"

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
	dryRunFlag        = "dry-run"
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
			&cli.StringSliceFlag{
				Name:  allowHazardsFlag,
				Usage: "hazardous operation types to allow (e.g. INDEX_BUILD, DELETES_DATA); may be repeated",
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  noSchemaDriftFlag,
				Usage: "skip check for schema drift before applying migrations",
				Value: false,
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  dryRunFlag,
				Usage: "preview migrations without applying them",
				Value: false,
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:    commandutil.Verbose,
				Aliases: []string{"v"},
				Usage:   "show migration SQL content in dry-run output",
				Value:   false,
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

			opts := []conduit.Option{conduit.WithRegistry(registry)}
			if cmd.Bool(noSchemaDriftFlag) {
				opts = append(opts, conduit.WithSkipSchemaDriftCheck())
			}

			if cmd.Bool(dryRunFlag) {
				opts = append(opts, conduit.WithExecutor(
					conduit.NewDryRunExecutor(os.Stdout, cmd.Bool(commandutil.Verbose)),
				))
			}

			migrator := conduit.NewMigrator(opts...)

			args := conduitcli.ApplyArgs{
				DatabaseURL:  url,
				Direction:    dir,
				Steps:        cmd.Int(stepsFlag),
				AllowHazards: cmd.StringSlice(allowHazardsFlag),
			}

			err = conduitcli.Apply(ctx, migrator, args)
			if err != nil && errors.Is(err, conduit.ErrHazardDetected) {
				return fmt.Errorf(
					"%w\n\nuse --%s <HAZARD_TYPE>... to allow specific hazard types",
					err,
					allowHazardsFlag,
				)
			}

			return err
		},
	}
}
