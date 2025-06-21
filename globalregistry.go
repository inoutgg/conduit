package conduit

import (
	"io/fs"

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

// Up registers an up migration function in the global registry.
func Up(up MigrateFunc) { globalRegistry.Up(up) }

// UpTx registers an up migration function that runs in a transaction in the global registry.
func UpTx(up MigrateFuncTx) { globalRegistry.UpTx(up) }

// Down registers a down migration function in the global registry.
func Down(down MigrateFunc) { globalRegistry.Down(down) }

// DownTx registers a down migration function that runs in a transaction in the global registry.
func DownTx(down MigrateFuncTx) { globalRegistry.DownTx(down) }

// FromFS registers SQL migrations from the provided filesystem in the global registry.
func FromFS(fsys fs.FS) {
	globalRegistry.FromFS(fsys)
}
