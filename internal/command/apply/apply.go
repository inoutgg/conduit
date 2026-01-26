package apply

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/direction"
)

//nolint:gochecknoglobals
var stepsFlag = "steps"

//nolint:revive // ignore naming convention.
type ApplyArgs struct {
	DatabaseURL string
	Direction   direction.Direction
	Steps       int
}

func NewCommand(migrator *conduit.Migrator) *cli.Command {
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

			args := ApplyArgs{
				DatabaseURL: url,
				Direction:   dir,
				Steps:       cmd.Int(stepsFlag),
			}

			return apply(ctx, migrator, args)
		},
	}
}

// apply applies a migration in the defined direction.
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
		Steps: args.Steps,
	})
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
