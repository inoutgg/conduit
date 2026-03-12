package conduitregistry

import (
	"context"
	"errors"
	"io/fs"
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

// Registry holds a set of parsed SQL migrations keyed by version and name.
type Registry struct {
	migrations map[string]*Migration
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{
		migrations: make(map[string]*Migration),
	}
}

// FromIOFS parses all .up.sql and .down.sql files under root in the given fs (io/fs)
// and returns a populated [Registry]. It panics if parsing fails.
func FromIOFS(fs fs.FS, root string) *Registry {
	return FromFS(afero.FromIOFS{FS: fs}, root)
}

// FromFS parses all .up.sql and .down.sql files under root in the given fs (afero.Fs)
// and returns a populated [Registry]. It panics if parsing fails.
func FromFS(fs afero.Fs, root string) *Registry {
	r := New()

	migrations := must.Must(parseSQLMigrationsFromFS(fs, root))
	for _, m := range migrations {
		r.migrations[m.migrationKey()] = m
	}

	return r
}

// Migrations returns a shallow copy of the registered migrations map.
func (r *Registry) Migrations() map[string]*Migration {
	return maps.Clone(r.migrations)
}
