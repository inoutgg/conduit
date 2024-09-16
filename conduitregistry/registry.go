package conduitregistry

import (
	"context"
	"io/fs"
	"maps"

	"github.com/jackc/pgx/v5"
)

// MigrateFunc applies either up or down migration.
type MigrateFunc func(context.Context, pgx.Tx) error

// Migration represents a single database migration.
type Migration interface {
	// Version returns a version of this migration.
	Version() int64

	// Name returns human-readable name of this migration.
	Name() string

	// Up applies schema changes to a database.
	Up(context.Context, pgx.Tx) error

	// Down rolls back schema changes previously applied to a database.
	Down(context.Context, pgx.Tx) error
}

// Registry is a local registry for the migration files.
//
// It is used to register both SQL file and dynamic (i.e., Go sourced)
// migrations.
type Registry struct {
	Namespace  string
	migrations map[int64]Migration
}

// New creates a new Registry.
func New(namespace string) *Registry {
	return &Registry{
		namespace,
		make(map[int64]Migration),
	}
}

// FromFS loads SQL migration files from the given fs.
// Typically fs is an embedded filesystem.
func (r *Registry) FromFS(fsys fs.FS) error {
	migrations, err := parseSQLMigrationsFromFS(fsys, ".")
	if err != nil {
		return err
	}

	for _, m := range migrations {
		r.migrations[m.Version()] = m
	}

	return nil
}

// Add adds Go migration with up and down functions to the registry.
func (r *Registry) Add(up, down MigrateFunc) error {
	m, err := newGoMigration(up, down)
	if err != nil {
		return err
	}

	r.migrations[m.Version()] = m

	return nil
}

// Migrations returns a cloned map of registered migrations.
func (r *Registry) Migrations() map[int64]Migration {
	return maps.Clone(r.migrations)
}
