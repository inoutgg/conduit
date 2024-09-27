// Package conduitmigrate exposes conduit migration script via conduit.Migrator.
package conduitmigrate

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/migrations"
)

type Migrate struct {
	migrator conduit.Migrator
}

type Config struct {
	Logger *slog.Logger
}

func New(config *Config) *Migrate {
	return &Migrate{
		migrator: conduit.NewMigrator(conduit.NewConfig(func(c *conduit.Config) {
			c.Logger = config.Logger
			c.Registry = migrations.Registry
		})),
	}
}

func (m *Migrate) Up(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionUp, conn)
	return err
}

// Down rolls back migration.
func (m *Migrate) Down(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionDown, conn)
	return err
}
