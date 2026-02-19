package conduit

import (
	"io/fs"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/conduitregistry"
)

const (
	// GlobalRegistryNamespace is the namespace of the global registry.
	//
	// The global registry is used by default by the Migrator when no
	// alternative registry is provided via Config.
	GlobalRegistryNamespace = "global"
)

//nolint:gochecknoglobals
var globalRegistry = conduitregistry.New(GlobalRegistryNamespace)

// FromFS registers SQL migrations from the provided filesystem in the global registry.
func FromFS(fs fs.FS, root string) {
	globalRegistry.FromFS(afero.FromIOFS{FS: fs}, root)
}
