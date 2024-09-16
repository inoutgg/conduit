package migrations

import (
	"embed"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/foundations/must"
)

var Registry *conduitregistry.Registry = conduitregistry.New("conduit/versioning")

//go:embed **.sql
var migrations embed.FS

func init() {
	must.Must1(Registry.FromFS(migrations))
}