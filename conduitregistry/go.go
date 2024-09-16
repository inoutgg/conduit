package conduitregistry

import (
	"context"
	"errors"
	"runtime"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/internal/version"
)

var _ Migration = (*goMigration)(nil)

type goMigration struct {
	version int64
	name    string

	up   MigrateFunc
	down MigrateFunc
}

func (m *goMigration) Version() int64 { return m.version }
func (m *goMigration) Name() string   { return m.name }

func (m *goMigration) Up(ctx context.Context, tx pgx.Tx) error   { return m.up(ctx, tx) }
func (m *goMigration) Down(ctx context.Context, tx pgx.Tx) error { return m.down(ctx, tx) }

func newGoMigration(up, down MigrateFunc) (*goMigration, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return nil, errors.New("conduit: failed to retrieve caller information")
	}

	info, err := version.ParseMigrationFilename(filename)
	if err != nil {
		return nil, err
	}

	return &goMigration{
		version: info.Version,
		name:    info.Name,
		up:      up,
		down:    down,
	}, nil
}
