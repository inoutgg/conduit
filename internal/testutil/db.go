package testutil

import (
	"fmt"
	"net"
	"strconv"
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

func ConnString(pool *pgxpool.Pool) string {
	cc := pool.Config().ConnConfig

	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cc.User, cc.Password, net.JoinHostPort(cc.Host, strconv.Itoa(int(cc.Port))), cc.Database,
	)
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
