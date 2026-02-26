package testregistry

import (
	"testing"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/testutil"
)

func NewRegistry(t *testing.T, files map[string]string) *conduitregistry.Registry {
	t.Helper()

	builder := testutil.NewMigrationsDirBuilder(t)
	for name, content := range files {
		builder.WithFile(name, content)
	}

	fs, _, dir := builder.Build()

	return conduitregistry.FromFS(fs, dir)
}
