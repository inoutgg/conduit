package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type NoopMigrator struct{}

func (NoopMigrator) Migrate(context.Context, *pgx.Conn) error { return nil }
func (NoopMigrator) Hash() string                             { return "noop" }

// TableExists checks whether a table with the given name exists in the database.
func TableExists(tb testing.TB, pool *pgxpool.Pool, name string) bool {
	tb.Helper()

	var exists bool

	err := pool.QueryRow(
		tb.Context(),
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
		name,
	).Scan(&exists)
	require.NoError(tb, err)

	return exists
}
