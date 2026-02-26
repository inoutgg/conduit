package apply

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/direction"
)

//nolint:gochecknoglobals
var (
	stepsFlag         = "steps"
	allowHazardsFlag  = "allow-hazards"
	noSchemaDriftFlag = "no-check-schema-drift"
)

//nolint:revive // ignore naming convention.
type ApplyArgs struct {
	DatabaseURL          string
	Direction            direction.Direction
	SkipSchemaDriftCheck bool
	AllowHazards         bool
	Steps                int
}

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "apply",
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:     flagname.DatabaseURL,
				Usage:    "database connection URL",
				Sources:  cli.EnvVars("CONDUIT_DATABASE_URL"),
				Required: true,
			},

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

			url := cmd.String(flagname.DatabaseURL)
			if url == "" {
				return fmt.Errorf("missing `%s' flag", flagname.DatabaseURL)
			}

			migrator := conduit.NewMigrator()

			args := ApplyArgs{
				DatabaseURL:          url,
				Direction:            dir,
				Steps:                cmd.Int(stepsFlag),
				AllowHazards:         cmd.Bool(allowHazardsFlag),
				SkipSchemaDriftCheck: cmd.Bool(noSchemaDriftFlag),
			}

			return apply(ctx, migrator, args)
		},
	}
}

func apply(
	ctx context.Context,
	migrator *conduit.Migrator,
	args ApplyArgs,
) error {
	conn, err := pgx.Connect(ctx, args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = migrator.Migrate(ctx, args.Direction, conn, &conduit.MigrateOptions{
		Steps:                args.Steps,
		AllowHazards:         args.AllowHazards,
		SkipSchemaDriftCheck: args.SkipSchemaDriftCheck,
	})
	if err != nil {
		if errors.Is(err, conduit.ErrHazardDetected) {
			return fmt.Errorf("%w\n\nuse --%s to proceed", err, allowHazardsFlag)
		}

		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
