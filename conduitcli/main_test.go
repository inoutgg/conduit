package conduitcli

import (
	"context"
	"os"
	"testing"
	"time"

	"go.segfaultmedaddy.com/pgxephemeraltest"
	"go.uber.org/goleak"

	"go.inout.gg/conduit/internal/testmigrator"
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/timegenerator"
)

// buildInfoStub is a test implementation of buildinfo.BuildInfo.
type buildInfoStub struct{}

func (buildInfoStub) Version() string { return "devel" }

//nolint:gochecknoglobals
var (
	poolFactory *pgxephemeraltest.PoolFactory
	timeGen                         = timegenerator.Stub{T: time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)}
	bi          buildinfo.BuildInfo = buildInfoStub{}
)

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
