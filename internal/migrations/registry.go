package migrations

import (
	"embed"

	"go.inout.gg/conduit/conduitregistry"
)

const RegistryNamespace = "inout/conduit"

// Conduit's own migrations scripts registry.
var Registry *conduitregistry.Registry = conduitregistry.New(RegistryNamespace) //nolint:gochecknoglobals

//go:embed **.sql
var migrations embed.FS

//nolint:gochecknoinits
func init() {
	Registry.FromFS(migrations)
}
