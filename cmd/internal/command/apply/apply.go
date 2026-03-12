package apply

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"time"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	yamlsrc "github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/pkg/stopwatch"
)

const (
	stepsFlag           = "steps"
	allowHazardsFlag    = "allow-hazards"
	skipSchemaDriftFlag = "skip-schema-drift-check"
	dryRunFlag          = "dry-run"
)

func NewCommand(
	fs afero.Fs,
	stdout io.Writer,
	stderr io.Writer,
	timer stopwatch.Stopwatch,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "apply",
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			//nolint:exhaustruct
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
				),
			},

			cmdutil.VerboseFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, err := direction.FromString(cmd.Args().First())
			if err != nil {
				return fmt.Errorf("failed to parse direction: %w", err)
			}

			migrationsDir := cmd.String(cmdutil.MigrationsDir)
			isDryRun := cmd.Bool(dryRunFlag)

			opts := []conduit.Option{
				conduit.WithRegistry(conduitregistry.FromFS(fs, migrationsDir)),
			}
			if cmd.Bool(skipSchemaDriftFlag) {
				opts = append(opts, conduit.WithSkipSchemaDriftCheck())
			}

			if isDryRun {
				opts = append(opts, conduit.WithExecutor(
					conduit.NewDryRunExecutor(stdout, cmd.Bool(cmdutil.Verbose)),
				))
			} else {
				opts = append(opts, conduit.WithExecutor(
					conduit.NewLiveExecutor(slog.Default(), timer),
				))
			}

			migrator := conduit.NewMigrator(opts...)

			args := conduitcli.ApplyArgs{
				DatabaseURL:  cmd.String(cmdutil.DatabaseURL),
				Direction:    dir,
				Steps:        cmd.Int(stepsFlag),
				AllowHazards: cmd.StringSlice(allowHazardsFlag),
			}

			seq, err := conduitcli.Apply(ctx, migrator, args)
			if err != nil {
				//nolint:wrapcheck
				return err
			}

			return displayResults(stderr, seq, dir, isDryRun)
		},
	}
}

func displayResults(
	w io.Writer,
	seq iter.Seq2[*conduit.MigrationResult, error],
	dir direction.Direction,
	isDryRun bool,
) error {
	var (
		n     int
		total time.Duration
	)

	for m, err := range seq {
		if err != nil {
			return err
		}

		n++
		total += m.DurationTotal

		switch {
		case isDryRun:
			fmt.Fprintf(w, "Pending %s_%s\n", m.Version.String(), m.Name)
		case dir == direction.DirectionDown:
			fmt.Fprintf(
				w, "Rolled back %s_%s (%s)\n",
				m.Version.String(), m.Name, formatDuration(m.DurationTotal),
			)
		default:
			fmt.Fprintf(
				w, "Applied %s_%s (%s)\n",
				m.Version.String(), m.Name, formatDuration(m.DurationTotal),
			)
		}
	}

	if n == 0 {
		if dir == direction.DirectionUp {
			fmt.Fprintln(w, "No pending migrations.")
		} else {
			fmt.Fprintln(w, "No migrations to roll back.")
		}

		return nil
	}

	fmt.Fprintln(w)

	switch {
	case isDryRun:
		fmt.Fprintf(w, "%d pending migrations (dry run)\n", n)
	case dir == direction.DirectionDown:
		fmt.Fprintf(w, "Rolled back %d migrations in %s\n", n, formatDuration(total))
	default:
		fmt.Fprintf(w, "Applied %d migrations in %s\n", n, formatDuration(total))
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	return d.Round(time.Millisecond).String()
}
