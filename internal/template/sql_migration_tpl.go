package template

import (
	"text/template"

	"go.inout.gg/foundations/must"
)

var SQLMigrationTemplate = must.Must(template.New("SQL Migration Template").Parse(`-- migration: {{.Version}}_{{.Name}}.sql

SELECT "up_{{.Version}}";

---- create above / drop below ----

SELECT "down_{{.Version}}";
`))
