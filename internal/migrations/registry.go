package migrations

import (
	"embed"

	"go.inout.gg/conduit/conduitregistry"
)

const RegistryNamespace = "inout/conduit"

var (
	Registry *conduitregistry.Registry = conduitregistry.New(RegistryNamespace)

	// Version of the very first migration in the conduit project.
	FirstMigrationVersion = 1726004714907
)

//go:embed **.sql
var migrations embed.FS

func init() {
	Registry.FromFS(migrations)
}
