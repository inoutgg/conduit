package conduit_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.segfaultmedaddy.com/pgxephemeraltest"
	"go.uber.org/goleak"

	"go.inout.gg/conduit/internal/migrations"
)

//nolint:gochecknoglobals
var poolFactory *pgxephemeraltest.PoolFactory

type migrator struct{}

func (migrator) Migrate(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, string(migrations.Schema))
	return fmt.Errorf("failed to migrate schema: %w", err)
}

func (migrator) Hash() string { return "bootstrap" }

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error

	poolFactory, err = pgxephemeraltest.NewPoolFactoryFromConnString(
		ctx,
		os.Getenv("TEST_DATABASE_URL"),
		&migrator{},
	)
	if err != nil {
		panic(err)
	}

	goleak.VerifyTestMain(m)
}
