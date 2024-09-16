// Package `conduit` implements an SQL migration functionality fully designed
// to be used via embedding in Go application, somewhat similar to Goose.
//
// The package doesn't support any SQL drivers other than pgx v5.
package conduit

import (
	"io/fs"

	"go.inout.gg/conduit/conduitregistry"
)

const (
	// Then namespace name of the global registry.
	// The global registry is used by default by the Migrator in case
	// when no alternative registry is provided via  Config.
	GlobalRegistryNamespace = "default"
)

var globalRegistry = conduitregistry.New(GlobalRegistryNamespace)

// Add registers a new migration with up and down Go functions in the global registry.
func Add(up, down MigrateFunc) error {
	return globalRegistry.Add(up, down)
}

// FromFS registers one or more SQL migrations from the fsys in the global registry.
func FromFS(fsys fs.FS) error {
	return globalRegistry.FromFS(fsys)
}
