package apply

import (
	"context"
	"os"
	"testing"

	"go.segfaultmedaddy.com/pgxephemeraltest"
	"go.uber.org/goleak"

	"go.inout.gg/conduit/internal/testmigrator"
)

//nolint:gochecknoglobals
var poolFactory *pgxephemeraltest.PoolFactory

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error

	poolFactory, err = pgxephemeraltest.NewPoolFactoryFromConnString(
		ctx,
		os.Getenv("TEST_DATABASE_URL"),
		testmigrator.ConduitMigrator,
	)
	if err != nil {
		panic(err)
	}

	goleak.VerifyTestMain(m)
}
