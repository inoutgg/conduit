package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"go.inout.gg/foundations/debug"
	"go.inout.gg/conduit/internal/version"
	"go.inout.gg/foundations/must"
)

var sqlTemplate = must.Must(template.New("conduit: SQL Migration Template").Parse(`-- migration: {{.Version}} {{.Name}}

SELECT "up_{{.Version}}";

---- create above / drop below ----

SELECT "down_{{.Version}}";
`))
var goTemplate = must.Must(template.New("conduit: Go Migration Template").Parse(`-- migration: {{.Version}} {{.Name}}

package migrations

import (
	"context"
)

func up{{.Version}}(ctx context.Context) error {
	return nil
}

func down{{.Version}}(ctx context.Context) error {
	return nil
}
`))

// CreateMigrationFile creates a migration file at [dir].
func CreateMigrationFile(ctx context.Context, dir string, ver int64, name string, ext string) error {
	dir = filepath.Clean(dir)
	filename := version.MigrationFilename(ver, name, ext)
	path := filepath.Join(dir, filename)

	// Ensure migration dir exists.
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("conduit: unable to create migrations folder %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("conduit: unable to create migration %s file: %w", path, err)
	}
	defer f.Close()

	info := templateInfo{
		Ext:     ext,
		Name:    name,
		Version: ver,
	}
	if err := writeTemplate(f, &info); err != nil {
		d("failed to write template: %w", err)
		return err
	}

	return nil
}

type templateInfo struct {
	Version int64
	Name    string
	Ext     string
	Package string
}

func writeTemplate(w io.Writer, info *templateInfo) error {
	d("Writing migration file: name=%s, ext=%s", info.Name, info.Ext)
	debug.Assert(info.Ext == "go" || info.Ext == "sql", "Unsupported extension: %s", info.Ext)
	var tpl *template.Template

	switch info.Ext {
	case "go":
		tpl = goTemplate
	case "sql":
		tpl = sqlTemplate
	default:
		panic("conduit: unsupported extension")
	}

	if err := tpl.Execute(w, info); err != nil {
		return fmt.Errorf("conduit: unable to write template: %w", err)
	}

	return nil
}
