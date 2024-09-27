package template

import (
	"text/template"

	"go.inout.gg/foundations/must"
)

var GoMigrationTemplate = must.Must(template.New("Go Migration Template").Parse(`// migration: {{.Version}}_{{.Name}}.go

package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
{{- if not .HasCustomRegistry}}
		"go.inout.gg/conduit"
{{- end}}
)

func init() {
{{- if .HasCustomRegistry}}
	Registry.Add(up{{.Version}}, down{{.Version}})
{{- else}}
	conduit.Add(up{{.Version}}, down{{.Version}})
{{- end}}
}

func up{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return nil
}

func down{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return nil
}
`))
