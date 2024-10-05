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
	Registry.UpTx(up{{.Version}})
	Regsitry.DownTx(down{{.Version}})
{{- else}}
	conduit.UpTx(up{{.Version}})
	conduit.DownTx(down{{.Version}})
{{- end}}
}

func up{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return nil
}

func down{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return nil
}
`))
