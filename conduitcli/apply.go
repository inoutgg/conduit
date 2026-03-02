// Package conduitcli provides high-level operations for the conduit CLI.
//
// Each function connects to a Postgres database and performs a single
// migration operation (apply, diff, dump, or init). They are intended
// to be called from CLI command handlers but can also be used
// programmatically.
package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/direction"
)

// ApplyArgs configures a migration apply operation.
type ApplyArgs struct {
	DatabaseURL  string
	Direction    direction.Direction
	AllowHazards []conduit.HazardType
	Steps        int
}

// Apply connects to the database and runs pending migrations in the given direction.
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
		Steps:        args.Steps,
		AllowHazards: args.AllowHazards,
	})
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
