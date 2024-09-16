package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"go.inout.gg/conduit/internal/version"
	"go.inout.gg/foundations/must"
)

var migrationsTempltae = must.Must(template.New("conduit: Go Migration Template").Parse(`-- migration: {{.Version}} {{.Name}}

package migrations

import (
	"context"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitmigrate"
)

var m{{.Version}} = conduitmigrate.New(nil, &migration.Config{})

func init() {
	conduit.Add(up{{.Version}}, down{{.Version}})
}

func up{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return m{{.Version}}.Up(ctx, tx)
}

func down{{.Version}}(ctx context.Context, tx pgx.Tx) error {
	return m{{.Version}}.Down(ctx, tx)
}
`))

type InitConfig struct {
	IncludeMigrationsSchema bool
}

func Init(ctx context.Context, dir string, config *InitConfig) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("conduit: unable to create migrations directory at %s: %w", dir, err)
	}

	if config.IncludeMigrationsSchema {
		dir = filepath.Clean(dir)
		filename := version.MigrationFilename(time.Now().UnixMilli(), "migrations", "go")
		filepath := filepath.Join(dir, filename)

		f, err := os.Create(filepath)
		if err != nil {
			return fmt.Errorf("conduit: failed to create migrations file: %w", err)
		}
		defer f.Close()
	}

	return nil
}
