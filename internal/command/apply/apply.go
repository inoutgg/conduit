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
var stepsFlagName = "steps"

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
				Name:  stepsFlagName,
				Usage: "maximum migrations steps",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return apply(ctx, cmd, migrator)
		},
	}
}

// apply applies a migration in the defined direction.
func apply(
	ctx context.Context,
	cmd *cli.Command,
	migrator *conduit.Migrator,
) error {
	dir, err := direction.FromString(cmd.Args().First())
	if err != nil {
		return fmt.Errorf("conduit: failed to parse direction: %w", err)
	}

	url := cmd.String(flagname.DatabaseURL)
	if url == "" {
		return fmt.Errorf("conduit: missing `%s' flag", flagname.DatabaseURL)
	}

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("conduit: failed to connect to database: %w", err)
	}

	opts := &conduit.MigrateOptions{
		Steps: cmd.Int(stepsFlagName),
	}

	_, err = migrator.Migrate(ctx, dir, conn, opts)
	if err != nil {
		return fmt.Errorf("conduit: failed to apply migrations: %w", err)
	}

	return nil
}
