package template

import (
	"text/template"

	"go.inout.gg/foundations/must"

	_ "embed"
)

var (
	//go:embed conduit_migration.go.tmpl
	conduitMigrationTemplate string

	//go:embed registry.go.tmpl
	registryTemplate string

	//go:embed migration.up.sql.tmpl
	sqlUpMigrationTemplate string

	//go:embed migration.down.sql.tmpl
	sqlDownMigrationTemplate string

	//go:embed migration.go.tmpl
	goMigrationTemplate string
)

var (
	ConduitMigrationTemplate *template.Template //nolint:gochecknoglobals
	RegistryTemplate         *template.Template //nolint:gochecknoglobals
	SQLUpMigrationTemplate   *template.Template //nolint:gochecknoglobals
	SQLDownMigrationTemplate *template.Template //nolint:gochecknoglobals
	GoMigrationTemplate      *template.Template //nolint:gochecknoglobals
)

//nolint:gochecknoinits
func init() {
	ConduitMigrationTemplate = must.Must(
		template.New("conduit: Conduit Migration Template").Parse(conduitMigrationTemplate),
	)
	RegistryTemplate = must.Must(template.New("conduit: Registry Template").Parse(registryTemplate))
	SQLUpMigrationTemplate = must.Must(
		template.New("conduit: SQL Up Migration Template").Parse(sqlUpMigrationTemplate),
	)
	SQLDownMigrationTemplate = must.Must(
		template.New("conduit: SQL Down Migration Template").Parse(sqlDownMigrationTemplate),
	)
	GoMigrationTemplate = must.Must(template.New("conduit: Go Migration Template").Parse(goMigrationTemplate))
}
