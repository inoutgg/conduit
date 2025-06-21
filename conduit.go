// Package `conduit` implements an SQL migration functionality fully designed
// to be used via embedding in Go application, somewhat similar to Goose.
//
// The package doesn't support any SQL drivers other than pgx v5.
package conduit

import (
	"go.inout.gg/foundations/debug"
)

//nolint:gochecknoglobals
var d = debug.Debuglog("conduit")
