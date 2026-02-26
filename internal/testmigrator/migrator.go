package testmigrator

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.segfaultmedaddy.com/pgxephemeraltest"

	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/pkg/sqlsplit"
)

var (
	_ pgxephemeraltest.Migrator = (*noopMigrator)(nil)
	_ pgxephemeraltest.Migrator = (*conduitMigrator)(nil)
)

//nolint:gochecknoglobals
var (
	NoopMigrator    = &noopMigrator{}
	ConduitMigrator = &conduitMigrator{}
)

type noopMigrator struct{}

func (*noopMigrator) Migrate(context.Context, *pgx.Conn) error { return nil }
func (*noopMigrator) Hash() string                             { return "noop" }

type conduitMigrator struct{}

func (*conduitMigrator) Migrate(ctx context.Context, conn *pgx.Conn) error {
	stmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return fmt.Errorf("failed to split schema: %w", err)
	}

	for _, stmt := range stmts {
		if stmt.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		_, err := conn.Exec(ctx, stmt.Content)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

func (*conduitMigrator) Hash() string { return "conduit" }
