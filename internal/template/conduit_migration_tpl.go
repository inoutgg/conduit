package template

import (
	"text/template"

	"go.inout.gg/foundations/must"
)

var ConduitMigrationTemplate = must.Must(template.New("conduit: Conduit Migration Template").Parse(`package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
{{- if not .HasCustomRegistry}}
	"go.inout.gg/conduit"
{{- end}}
	"go.inout.gg/conduit/conduitmigrate"
)

var m{{.Version}} = conduitmigrate.New(&conduitmigrate.Config{})

func init() {
{{- if .HasCustomRegistry}}
	Registry.Add(up{{.Version}}, down{{.Version}})
{{- else}}
	conduit.Add(up{{.Version}}, down{{.Version}})
{{- end}}
}

func up{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return m{{.Version}}.Up(ctx, tx)
}

func down{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return m{{.Version}}.Down(ctx, tx)
}
`))
