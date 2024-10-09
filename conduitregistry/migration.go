package conduitregistry

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/foundations/debug"
)

var emptyMigrateFunc = &migrateFunc{
	fn:  func(_ context.Context, _ *pgx.Conn) error { return nil },
	fnx: func(_ context.Context, _ pgx.Tx) error { return nil },
}

type migrateFunc struct {
	fn   MigrateFunc
	fnx  MigrateFuncTx
	inTx bool
}

// Migration represents a single database migration.
type Migration struct {
	version int64
	name    string

	up   *migrateFunc
	down *migrateFunc
}

// InTx tests whether migration should run in transation for given direction.
func (m *Migration) UseTx(dir direction.Direction) (bool, error) {
	switch dir {
	case direction.DirectionUp:
		return m.up.inTx, nil
	case direction.DirectionDown:
		return m.down.inTx, nil
	}

	return false, direction.ErrUnknownDirection
}

// Version returns the version of this migration.
func (m *Migration) Version() int64 { return m.version }

// Name returns the name of this migration.
func (m *Migration) Name() string { return m.name }

// Apply executes the transaction in given dir direction.
func (m *Migration) Apply(ctx context.Context, dir direction.Direction, conn *pgx.Conn, tx pgx.Tx) error {
	debug.Assert(conn != nil, "conduit: expected conn to be defined")

	switch dir {
	case direction.DirectionUp:
		return m.migrateUp(ctx, conn, tx)
	case direction.DirectionDown:
		return m.migrateDown(ctx, conn, tx)
	}

	return direction.ErrUnknownDirection
}

func (m *Migration) migrateDown(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) error {
	if m.down.inTx {
		if tx == nil {
			return UndefinedTxErr
		}
		return m.down.fnx(ctx, tx)
	}

	return m.down.fn(ctx, conn)
}

func (m *Migration) migrateUp(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) error {
	if m.down.inTx {
		if tx == nil {
			return UndefinedTxErr
		}
		return m.up.fnx(ctx, tx)
	}

	return m.up.fn(ctx, conn)
}
