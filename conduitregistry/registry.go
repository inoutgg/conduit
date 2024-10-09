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
	UndefinedTxErr    = errors.New("conduit: tx must be defined")
	EmptyMigrationErr = errors.New("conduit: migration is empty")
	UpExistsErr       = errors.New("conduit: up migration already registered")
	DownExistsErr     = errors.New("conduit: down migration already registered")
)

// MigrateFunc applies an up or down migration.
type MigrateFunc func(context.Context, *pgx.Conn) error

// MigrateFuncTx applies an up or down migration within a transaction.
type MigrateFuncTx func(context.Context, pgx.Tx) error

// Registry stores migration files, both SQL and Go-sourced.
type Registry struct {
	Namespace  string
	migrations map[int64]*Migration
}

// New creates a new Registry with the given namespace.
func New(namespace string) *Registry {
	return &Registry{
		namespace,
		make(map[int64]*Migration),
	}
}

// FromFS loads SQL migration files from the given filesystem.
//
// SQL migrations run in transaction mode by default.
// To disable transactions, add `---- disable-tx ----` comment in the SQL.
// This comment applies to the current migration section (up or down).
// For down migrations, place the comment below the `---- create above / drop below ----` separator.
func (r *Registry) FromFS(fsys fs.FS) {
	migrations := must.Must(parseSQLMigrationsFromFS(fsys, "."))
	for _, m := range migrations {
		r.migrations[m.Version()] = m
	}
}

// Up registers a Go function for up migration.
func (r *Registry) Up(up MigrateFunc) {
	m := must.Must(r.goMigration())
	if m.up != nil {
		panic(UpExistsErr)
	}
	m.up = &migrateFunc{fn: up, inTx: false}
}

// UpTx registers a Go function for up migration within a transaction.
func (r *Registry) UpTx(up MigrateFuncTx) {
	m := must.Must(r.goMigration())
	if m.up != nil {
		panic(UpExistsErr)
	}
	m.up = &migrateFunc{fnx: up, inTx: true}
}

// Down registers a Go function for down migration.
func (r *Registry) Down(down MigrateFunc) {
	m := must.Must(r.goMigration())
	if m.down != nil {
		panic(DownExistsErr)
	}
	m.down = &migrateFunc{fn: down, inTx: false}
}

// DownTx registers a Go function for down migration within a transaction.
func (r *Registry) DownTx(down MigrateFuncTx) {
	m := must.Must(r.goMigration())
	if m.down != nil {
		panic(DownExistsErr)
	}
	m.down = &migrateFunc{fnx: down, inTx: true}
}

// goMigration creates a new Migration from a Go registration function.
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

// CloneMigrations returns a copy of the registered migrations map.
func (r *Registry) CloneMigrations() map[int64]*Migration {
	return maps.Clone(r.migrations)
}
