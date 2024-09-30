package conduitregistry

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"runtime"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/internal/version"
	"go.inout.gg/foundations/must"
)

var (
	UndefinedTxErr    = errors.New("conduit: expected tx to be defined")
	EmptyMigrationErr = errors.New("conduit: empty migration")
	UpExistsErr       = errors.New("conduit: up migration is already registered")
	DownExistsErr     = errors.New("conduit: down migration is already registered")
)

// MigrateFunc applies either up or down migration.
type MigrateFunc func(context.Context, *pgx.Conn) error

// MigrateFuncTx applies either up or down migration in transaction.
type MigrateFuncTx func(context.Context, pgx.Tx) error

// Registry is a local registry for the migration files.
//
// It is used to register both SQL file and dynamic (i.e., Go sourced)
// migrations.
type Registry struct {
	Namespace  string
	migrations map[int64]*Migration
}

// New creates a new Registry.
func New(namespace string) *Registry {
	return &Registry{
		namespace,
		make(map[int64]*Migration),
	}
}

// FromFS loads SQL migration files from the given fs.
// Typically fs is an embedded filesystem.
func (r *Registry) FromFS(fsys fs.FS) {
	migrations := must.Must(parseSQLMigrationsFromFS(fsys, "."))
	for _, m := range migrations {
		r.migrations[m.Version()] = m
	}
}

// Up adds Go migration to registry to run on migration rolling.
func (r *Registry) Up(up MigrateFunc) {
	m := must.Must(r.goMigration())

	if m.up != nil {
		panic(UpExistsErr)
	}

	m.up = &migrateFunc{fn: up, inTx: false}
}

// UpTx adds Go migration to registry to run in transaction on migration rolling.
func (r *Registry) UpTx(up MigrateFuncTx) {
	m := must.Must(r.goMigration())

	if m.up != nil {
		panic(UpExistsErr)
	}

	m.up = &migrateFunc{fnx: up, inTx: true}
}

// Down adds Go migration to registry to run on migration rolling back.
func (r *Registry) Down(down MigrateFunc) {
	m := must.Must(r.goMigration())

	if m.down != nil {
		panic(DownExistsErr)
	}

	m.down = &migrateFunc{fn: down, inTx: false}
}

// DownTx adds Go migration to registry to run in transaction on migration rolling back.
func (r *Registry) DownTx(down MigrateFuncTx) {
	m := must.Must(r.goMigration())

	if m.down != nil {
		panic(DownExistsErr)
	}

	m.down = &migrateFunc{fnx: down, inTx: true}
}

// goMigration makes a new migration from a Go registration function.
func (r *Registry) goMigration() (*Migration, error) {
	_, filename, _, ok := runtime.Caller(2)
	if !ok {
		return nil, errors.New("conduit: failed to retrieve caller information")
	}

	info, err := version.ParseMigrationFilename(filename)
	if err != nil {
		return nil, err
	}

	if val, ok := r.migrations[info.Version]; ok {
		return val, nil
	}

	migration := &Migration{
		version: info.Version,
		name:    info.Name,
		up:      nil,
		down:    nil,
	}
	r.migrations[migration.version] = migration

	return migration, nil
}

// Migrations returns a cloned map of registered migrations.
func (r *Registry) Migrations() map[int64]*Migration {
	return maps.Clone(r.migrations)
}
