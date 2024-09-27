// Package `conduit` implements an SQL migration functionality fully designed
// to be used via embedding in Go application, somewhat similar to Goose.
//
// The package doesn't support any SQL drivers other than pgx v5.
package conduit

import (
	"io/fs"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/foundations/debug"
)

const (
	// Then namespace name of the global registry.
	// The global registry is used by default by the Migrator in case
	// when no alternative registry is provided via  Config.
	GlobalRegistryNamespace = "default"
)

var d = debug.Debuglog("conduit: conduit")

var globalRegistry = conduitregistry.New(GlobalRegistryNamespace)

func Up(up MigrateFunc) error         { return globalRegistry.Up(up) }
func UpTx(up MigrateFuncTx) error     { return globalRegistry.UpTx(up) }
func Down(down MigrateFunc) error     { return globalRegistry.Down(down) }
func DownTx(down MigrateFuncTx) error { return globalRegistry.DownTx(down) }

// FromFS registers one or more SQL migrations from the fsys in the global registry.
func FromFS(fsys fs.FS) error {
	return globalRegistry.FromFS(fsys)
}
