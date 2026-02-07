package migrations

import (
	"embed"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/conduitregistry"
)

const RegistryNamespace = "inout/conduit"

// Registry is a global registry for migrations scripts that is used by
// conduit by default.
var Registry *conduitregistry.Registry = conduitregistry.New(RegistryNamespace) //nolint:gochecknoglobals

//go:embed **.sql
var migrations embed.FS

//nolint:gochecknoinits
func init() {
	Registry.FromFS(afero.FromIOFS{FS: migrations}, ".")
}
