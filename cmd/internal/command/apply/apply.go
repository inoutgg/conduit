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
	"go.inout.gg/conduit/cmd/internal/config"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/direction"
)

const (
	stepsFlag           = "steps"
	allowHazardsFlag    = "allow-hazards"
	skipSchemaDriftFlag = "skip-schema-drift-check"
	dryRunFlag          = "dry-run"
)

func NewCommand(fs afero.Fs, cfg *config.Config) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "apply",
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			commandutil.DatabaseURLFlag(),
			commandutil.MigrationsDirFlag(),

			//nolint:exhaustruct
			&cli.IntFlag{
				Name:    stepsFlag,
				Usage:   "maximum migrations steps",
				Sources: cli.EnvVars("CONDUIT_STEPS"),
			},

			//nolint:exhaustruct
			&cli.StringSliceFlag{
				Name:    allowHazardsFlag,
				Usage:   "hazardous operation types to allow (e.g. INDEX_BUILD, DELETES_DATA); may be repeated",
				Sources: cli.EnvVars("CONDUIT_ALLOW_HAZARDS"),
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:    skipSchemaDriftFlag,
				Usage:   "skip check for schema drift before applying migrations",
				Sources: cli.EnvVars("CONDUIT_SKIP_SCHEMA_DRIFT_CHECK"),
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:    dryRunFlag,
				Usage:   "preview migrations without applying them",
				Sources: cli.EnvVars("CONDUIT_DRY_RUN"),
			},

			commandutil.VerboseFlag("show migration SQL content in dry-run output"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, err := direction.FromString(cmd.Args().First())
			if err != nil {
				return fmt.Errorf("failed to parse direction: %w", err)
			}

			url := commandutil.StringOr(cmd, commandutil.DatabaseURL, cfg.Database.URL)
			if url == "" {
				return fmt.Errorf("missing `%s' flag", commandutil.DatabaseURL)
			}

			dirPath, _ := config.FilePath(cfg.Migrations.Dir)
			migrationsDir := commandutil.StringOr(cmd, commandutil.MigrationsDir, dirPath)
			allowHazards := commandutil.StringSliceOr(cmd, allowHazardsFlag, cfg.Apply.AllowHazards)
			skipSchemaDrift := commandutil.BoolOr(cmd, skipSchemaDriftFlag, cfg.Apply.SkipSchemaDriftCheck)
			verbose := commandutil.BoolOr(cmd, commandutil.Verbose, cfg.Verbose)

			opts := []conduit.Option{conduit.WithRegistry(
				conduitregistry.FromFS(fs, migrationsDir),
			)}
			if skipSchemaDrift {
				opts = append(opts, conduit.WithSkipSchemaDriftCheck())
			}

			if cmd.Bool(dryRunFlag) {
				opts = append(opts, conduit.WithExecutor(
					conduit.NewDryRunExecutor(os.Stdout, verbose),
				))
			}

			migrator := conduit.NewMigrator(opts...)

			args := conduitcli.ApplyArgs{
				DatabaseURL:  url,
				Direction:    dir,
				Steps:        cmd.Int(stepsFlag),
				AllowHazards: allowHazards,
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
