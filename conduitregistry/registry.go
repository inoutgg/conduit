package conduitregistry

import (
	"context"
	"errors"
	"maps"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"
	"go.inout.gg/foundations/must"
)

var (
	ErrEmptyMigration = errors.New("conduit: migration is empty")
	ErrUpExists       = errors.New("conduit: up migration already registered")
	ErrDownExists     = errors.New("conduit: down migration already registered")
)

type (
	applyFunc   func(context.Context, *pgx.Conn) error
	applyFuncTx func(context.Context, pgx.Tx) error
)

// Registry stores SQL migration files.
type Registry struct {
	migrations map[string]*Migration
}

func New() *Registry {
	return &Registry{
		migrations: make(map[string]*Migration),
	}
}

// FromFS loads SQL migration files from the given filesystem.
//
// SQL migrations run in transaction mode by default.
// To disable transactions, add `---- disable-tx ----` comment in the SQL.
func FromFS(fs afero.Fs, root string) *Registry {
	r := New()

	migrations := must.Must(parseSQLMigrationsFromFS(fs, root))
	for _, m := range migrations {
		r.migrations[m.migrationKey()] = m
	}

	return r
}

func (r *Registry) CloneMigrations() map[string]*Migration {
	return maps.Clone(r.migrations)
}
