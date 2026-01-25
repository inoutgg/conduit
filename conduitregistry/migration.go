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
	fn    MigrateFunc
	fnx   MigrateFuncTx
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
	if m.down.useTx {
		if tx == nil {
			return ErrUndefinedTx
		}

		return m.down.fnx(ctx, tx)
	}

	return m.down.fn(ctx, conn)
}

func (m *Migration) migrateUp(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) error {
	if m.down.useTx {
		if tx == nil {
			return ErrUndefinedTx
		}

		return m.up.fnx(ctx, tx)
	}

	return m.up.fn(ctx, conn)
}
