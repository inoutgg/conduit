package migrations

import (
	"embed"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/conduitregistry"
)

// Registry is a global registry for migrations scripts that is used by
// conduit by default.
var Registry *conduitregistry.Registry = conduitregistry.New() //nolint:gochecknoglobals

//go:embed 20250629171951_initial_schema.up.sql
var InitialSchema []byte

//go:embed **.sql
var migrations embed.FS

//nolint:gochecknoinits
func init() {
	Registry.FromFS(afero.FromIOFS{FS: migrations}, ".")
}
