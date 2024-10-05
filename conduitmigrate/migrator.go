// Package conduitregistry provides a Go API for running Conduit's own migrations via API.
//
// By default, when initializing a new migration project using the Conduit CLI,
// the migrations folder will contain an automatically generated migration that
// utilizes this API.
package conduitmigrate

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/migrations"
)

// Migrator represents a database migrator that applies Conduit's own migrations.
//
// It wraps up the conduit.Migrator and exposes Up and Down methods for rolling up
// and back migrations respectively.
type Migrator struct {
	migrator conduit.Migrator
}

type Config struct {
	Logger *slog.Logger
}

// config can be a nil.
func New(config *Config) *Migrator {
	return &Migrator{
		migrator: conduit.NewMigrator(conduit.NewConfig(func(c *conduit.Config) {
			if config != nil {
				c.Logger = config.Logger
			}
			c.Registry = migrations.Registry
		})),
	}
}

// Up roll ups Conduit migration.
func (m *Migrator) Up(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionUp, conn, nil)
	return err
}

func (m *Migrator) Down(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionDown, conn, nil)
	return err
}
