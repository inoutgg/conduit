// Package conduitmigrate provides a Go API for running Conduit's own migrations via API.
//
// By default, when initializing a new migration project using the Conduit CLI,
// the migrations folder will contain an automatically generated migration that
// utilizes this API.
//
// The migration is necessary for conduit to track list of applied and unapplied migrations.
package conduitmigrate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/migrations"
)

// Migrator represents a database migrator that applies conduit's own migrations.
//
// It wraps up the conduit.Migrator and exposes Up and Down methods for rolling up
// and back migrations respectively.
type Migrator struct {
	migrator *conduit.Migrator
}

type Config struct {
	Logger *slog.Logger
}

// New creates a new migrator for conduit own migrations.
//
// It is can be used as part of another migrator, or as a standalone.
// It primarily rolls up and down conduit's own utilitary migrations necessary
// for proper functioning of the conduit framework.
//
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

// Up applies conduit migration.
func (m *Migrator) Up(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionUp, conn, nil)
	if err != nil {
		return fmt.Errorf("conduit: failed to apply migration: %w", err)
	}

	return nil
}

// Down rolls back conduit migration.
func (m *Migrator) Down(ctx context.Context, conn *pgx.Conn) error {
	_, err := m.migrator.Migrate(ctx, conduit.DirectionDown, conn, nil)
	if err != nil {
		return fmt.Errorf("conduit: failed to rollback migration: %w", err)
	}

	return nil
}
