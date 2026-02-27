package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/direction"
)

type ApplyArgs struct {
	DatabaseURL          string
	Direction            direction.Direction
	SkipSchemaDriftCheck bool
	AllowHazards         bool
	Steps                int
}

func Apply(
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
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
