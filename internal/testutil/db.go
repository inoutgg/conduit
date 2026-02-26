package testutil

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/pkg/sqlsplit"
)

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

func Exec(tb testing.TB, pool *pgxpool.Pool, sql string) {
	tb.Helper()

	stmts, err := sqlsplit.Split([]byte(sql))
	require.NoError(tb, err)

	for _, s := range stmts {
		if s.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		_, err := pool.Exec(tb.Context(), s.Content)
		require.NoError(tb, err)
	}
}
