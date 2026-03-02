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
	fn:      func(_ context.Context, _ *pgx.Conn) error { return nil },
	fnx:     func(_ context.Context, _ pgx.Tx) error { return nil },
	hazards: nil,
	content: "",
	useTx:   false,
}

// Hazard represents a hazardous operation detected in a migration.
type Hazard struct {
	Type    string
	Message string
}

type migrateFunc struct {
	fn      applyFunc
	fnx     applyFuncTx
	content string
	hazards []Hazard
	useTx   bool
}

// Migration represents a single database migration.
type Migration struct {
	up      *migrateFunc
	down    *migrateFunc
	version version.Version
	name    string
}

// UseTx reports whether the migration should run inside a transaction
// for the given direction.
func (m *Migration) UseTx(dir direction.Direction) (bool, error) {
	switch dir {
	case direction.DirectionUp:
		return m.up.useTx, nil
	case direction.DirectionDown:
		return m.down.useTx, nil
	}

	return false, direction.ErrUnknownDirection
}

func (m *Migration) Version() version.Version { return m.version }

func (m *Migration) Name() string { return m.name }

// Content returns the raw SQL content for the given direction.
func (m *Migration) Content(dir direction.Direction) string {
	switch dir {
	case direction.DirectionUp:
		return m.up.content
	case direction.DirectionDown:
		return m.down.content
	}

	return ""
}

// Hazards returns the hazardous operations detected in the migration
// for the given direction. Returns nil when no hazards are present.
func (m *Migration) Hazards(dir direction.Direction) []Hazard {
	switch dir {
	case direction.DirectionUp:
		return m.up.hazards
	case direction.DirectionDown:
		return m.down.hazards
	}

	return nil
}

// Apply executes the migration on a bare connection without a transaction.
// Use this for migrations that cannot run inside a transaction
// (e.g. CREATE INDEX CONCURRENTLY).
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

// ApplyTx executes the migration within the provided transaction.
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

func (m *Migration) migrationKey() string { return migrationKey(m.version, m.name) }

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
