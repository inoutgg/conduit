package template

import (
	"text/template"

	"go.inout.gg/foundations/must"

	_ "embed"
)

var (
	//go:embed migration.up.sql.tmpl
	sqlUpMigrationTemplate string

	//nolint:gochecknoglobals
	SQLUpMigrationTemplate *template.Template
)

//nolint:gochecknoinits
func init() {
	SQLUpMigrationTemplate = must.Must(
		template.New("conduit: SQL Up Migration Template").Parse(sqlUpMigrationTemplate),
	)
}
