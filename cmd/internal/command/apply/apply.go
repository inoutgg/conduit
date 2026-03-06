package apply

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	yamlsrc "github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/internal/direction"
)

const (
	stepsFlag           = "steps"
	allowHazardsFlag    = "allow-hazards"
	skipSchemaDriftFlag = "skip-schema-drift-check"
	dryRunFlag          = "dry-run"
)

func NewCommand(fs afero.Fs, src altsrc.Sourcer) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "apply",
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			cmdutil.DatabaseURLFlag(src),
			cmdutil.MigrationsDirFlag(src),

			//nolint:exhaustruct
			&cli.IntFlag{
				Name:  stepsFlag,
				Usage: "maximum migrations steps",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_STEPS"),
					yamlsrc.YAML("apply.steps", src),
				),
			},

			//nolint:exhaustruct
			&cli.StringSliceFlag{
				Name:  allowHazardsFlag,
				Usage: "hazardous operation types to allow (e.g. INDEX_BUILD, DELETES_DATA); may be repeated",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_ALLOW_HAZARDS"),
					yamlsrc.YAML("apply.allow-hazards", src),
				),
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  skipSchemaDriftFlag,
				Usage: "skip check for schema drift before applying migrations",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_SKIP_SCHEMA_DRIFT_CHECK"),
					yamlsrc.YAML("apply.skip-schema-drift-check", src),
				),
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  dryRunFlag,
				Usage: "preview migrations without applying them",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_DRY_RUN"),
					yamlsrc.YAML("apply.dry-run", src),
				),
			},

			cmdutil.VerboseFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, err := direction.FromString(cmd.Args().First())
			if err != nil {
				return fmt.Errorf("failed to parse direction: %w", err)
			}

			dbURL := cmd.String(cmdutil.DatabaseURL)
			if dbURL == "" {
				return fmt.Errorf("missing required flag: --%s", cmdutil.DatabaseURL)
			}

			migrationsDir := cmd.String(cmdutil.MigrationsDir)

			opts := []conduit.Option{conduit.WithRegistry(
				conduitregistry.FromFS(fs, migrationsDir),
			)}
			if cmd.Bool(skipSchemaDriftFlag) {
				opts = append(opts, conduit.WithSkipSchemaDriftCheck())
			}

			if cmd.Bool(dryRunFlag) {
				opts = append(opts, conduit.WithExecutor(
					conduit.NewDryRunExecutor(os.Stdout, cmd.Bool(cmdutil.Verbose)),
				))
			}

			migrator := conduit.NewMigrator(opts...)

			args := conduitcli.ApplyArgs{
				DatabaseURL:  dbURL,
				Direction:    dir,
				Steps:        cmd.Int(stepsFlag),
				AllowHazards: cmd.StringSlice(allowHazardsFlag),
			}

			return conduitcli.Apply(ctx, migrator, args)
		},
	}
}
