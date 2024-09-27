package template

import (
	"text/template"

	"go.inout.gg/foundations/must"
)

var RegistryTemplate = must.Must(template.New("Registry Template").Parse(`package migrations

import (
	"embed"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/foundations/must"
)

var Registry = conduitregistry.New({{.Namespace}})

//go:embed **.sql
var migrationFS embed.FS

func init() {
	must.Must1(Registry.FromFS(migrationFS))
}
`))
