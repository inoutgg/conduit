package create

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// startEphemeralPostgres starts an ephemeral PostgreSQL container using
// testcontainers and returns a connection configuration.
func startEphemeralPostgres(
	ctx context.Context,
	pgVersion string,
) (*pgxpool.Config, func(context.Context) error, error) {
	image := fmt.Sprintf("postgres:%s-alpine", pgVersion)

	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase("conduit"),
		postgres.WithUsername("conduit"),
		postgres.WithPassword("conduit"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return poolConfig, func(ctx context.Context) error {
		return container.Terminate(ctx)
	}, nil
}
