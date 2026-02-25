package conduit

import (
	"io/fs"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/conduitregistry"
)

//nolint:gochecknoglobals
var globalRegistry = conduitregistry.New()

// FromFS registers SQL migrations from the provided filesystem in the global registry.
func FromFS(fs fs.FS, root string) {
	globalRegistry = conduitregistry.FromFS(afero.FromIOFS{FS: fs}, root)
}
