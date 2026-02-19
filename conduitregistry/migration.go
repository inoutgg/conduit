package conduitregistry

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/foundations/debug"

	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/pkg/version"
)

//nolint:gochecknoglobals
var emptyMigrateFunc = &migrateFunc{
	fn:    func(_ context.Context, _ *pgx.Conn) error { return nil },
	fnx:   func(_ context.Context, _ pgx.Tx) error { return nil },
	useTx: false,
}

type migrateFunc struct {
	fn    applyFunc
	fnx   applyFuncTx
	useTx bool
}

// Migration represents a single database migration.
type Migration struct {
	up      *migrateFunc
	down    *migrateFunc
	version version.Version
	name    string
}

// UseTx tests whether migration should run in transition for given direction.
func (m *Migration) UseTx(dir direction.Direction) (bool, error) {
	switch dir {
	case direction.DirectionUp:
		return m.up.useTx, nil
	case direction.DirectionDown:
		return m.down.useTx, nil
	}

	return false, direction.ErrUnknownDirection
}

// Version returns the version of this migration.
func (m *Migration) Version() version.Version { return m.version }

// Name returns the name of this migration.
func (m *Migration) Name() string { return m.name }

// Apply executes the migration in a given dir direction.
func (m *Migration) Apply(ctx context.Context, dir direction.Direction, conn *pgx.Conn) error {
	debug.Assert(conn != nil, "expected conn to be defined")

	switch dir {
	case direction.DirectionUp:
		return m.migrateUp(ctx, conn, nil)
	case direction.DirectionDown:
		return m.migrateDown(ctx, conn, nil)
	}

	return direction.ErrUnknownDirection
}

// ApplyTx executes the migration in a given dir direction in transaction.
func (m *Migration) ApplyTx(ctx context.Context, dir direction.Direction, tx pgx.Tx) error {
	debug.Assert(tx != nil, "expected tx to be defined")

	switch dir {
	case direction.DirectionUp:
		return m.migrateUp(ctx, nil, tx)
	case direction.DirectionDown:
		return m.migrateDown(ctx, nil, tx)
	}

	return direction.ErrUnknownDirection
}

func (m *Migration) migrateDown(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) error {
	if m.down.useTx {
		debug.Assert(conn == nil && tx != nil, "expected only tx to be defined")

		return m.down.fnx(ctx, tx)
	}

	debug.Assert(conn != nil && tx == nil, "expected only conn to be defined")

	return m.down.fn(ctx, conn)
}

func (m *Migration) migrateUp(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) error {
	if m.up.useTx {
		debug.Assert(conn == nil && tx != nil, "expected only tx to be defined")

		return m.up.fnx(ctx, tx)
	}

	debug.Assert(conn != nil && tx == nil, "expected only conn to be defined")

	return m.up.fn(ctx, conn)
}
