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
	migrator conduit.MigratorTx
}

type Config struct {
	Logger *slog.Logger
}

func New(config *Config) *Migrate {
	return &Migrate{
		migrator: conduit.NewMigratorTx(&conduit.Config{
			Logger:   config.Logger,
			Registry: migrations.Registry,
		}),
	}
}

// 
func (m *Migrate) Up(ctx context.Context, tx pgx.Tx) error {
	return m.migrator.MigrateTx(ctx, conduit.DirectionUp, tx)
}

// Down rolls back migration.
func (m *Migrate) Down(ctx context.Context, tx pgx.Tx) error {
	return m.migrator.MigrateTx(ctx, conduit.DirectionDown, tx)
}
