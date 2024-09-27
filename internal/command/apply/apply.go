package apply

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"
	"go.inout.gg/conduit"
)

func NewCommand(migrator conduit.Migrator) *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Args:  true,
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{},
		Action: func(ctx *cli.Context) error {
			return apply(ctx, migrator)
		},
	}
}

// apply applies a migration in the defined direction.
func apply(
	ctx *cli.Context,
	migrator conduit.Migrator,
) error {
	dir := ctx.Args().First()
	if dir == "" {
		return fmt.Errorf("conduit: missing `direction\" argument")
	}

	url := ctx.String("database-url")
	if url == "" {
		return fmt.Errorf("conduit: missing `database-url\" flag.")
	}

	direction, err := stringToDirection(dir)
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx.Context, url)
	if err != nil {
		return err
	}
	result, err := migrator.Migrate(ctx.Context, direction, conn)
	if err != nil {
		return err
	}

	for _, r := range result.MigrationResults {
		print(r.Name, r.DurationTotal, r.Namespace, "\n")
	}

	return nil
}

func stringToDirection(str string) (conduit.Direction, error) {
	switch strings.TrimSpace(str) {
	case string(conduit.DirectionUp):
		return conduit.DirectionUp, nil
	case string(conduit.DirectionDown):
		return conduit.DirectionDown, nil
	}

	return "", fmt.Errorf(
		"conduit: invalid direction, expected \"up\" or \"down\", received: %s",
		str,
	)
}
