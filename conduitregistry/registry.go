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
	ErrEmptyMigration = errors.New("migration is empty")
	ErrUpExists       = errors.New("up migration already registered")
	ErrDownExists     = errors.New("down migration already registered")
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
// SQL migrations run outside a transaction by default.
// To enable transactions, add `---- enable-tx ----` comment in the SQL.
func FromFS(fs afero.Fs, root string) *Registry {
	r := New()

	migrations := must.Must(parseSQLMigrationsFromFS(fs, root))
	for _, m := range migrations {
		r.migrations[m.migrationKey()] = m
	}

	return r
}

// Migrations returns a shallow copy of the registered migrations map.
// The keys are composite strings of "<version>_<name>".
func (r *Registry) Migrations() map[string]*Migration {
	return maps.Clone(r.migrations)
}
