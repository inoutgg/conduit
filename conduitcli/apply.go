// Package conduitcli provides high-level operations for the conduit CLI.
package conduitcli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/direction"
)

// ApplyArgs configures an [Apply] operation.
type ApplyArgs struct {
	DatabaseURL  string
	Direction    direction.Direction
	AllowHazards []conduit.HazardType
	Steps        int
}

// Apply connects to the database and applies migrations in the configured
// direction.
func Apply(
	ctx context.Context,
	migrator *conduit.Migrator,
	args ApplyArgs,
) (*conduit.MigrateResult, error) {
	conn, err := pgx.Connect(ctx, args.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	result, err := migrator.Migrate(ctx, args.Direction, conn, &conduit.MigrateOptions{
		Steps:        args.Steps,
		AllowHazards: args.AllowHazards,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return result, nil
}
