package testutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.segfaultmedaddy.com/pgxephemeraltest"
)

var _ pgxephemeraltest.Migrator = (*noopMigrator)(nil)

//nolint:gochecknoglobals
var NoopMigrator = &noopMigrator{}

type noopMigrator struct{}

func (*noopMigrator) Migrate(context.Context, *pgx.Conn) error { return nil }
func (*noopMigrator) Hash() string                             { return "noop" }
