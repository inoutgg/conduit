package create

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"go.inout.gg/conduit/internal/command/root"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/version"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create a new migration file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ext",
				Usage: "migration file extension (values: \"go\", \"sql\")",
				Value: "sql",
			},
		},
		Args:   true,
		Action: create,
	}
}

func create(ctx *cli.Context) error {
	dir, err := root.MigrationDir(ctx)
	if err != nil {
		return err
	}

	// Ensure migration dir exists.
	if !exists(dir) {
		return fmt.Errorf(
			"conduit: migrations directory does not exist. Try to initialise it first.",
		)
	}

	name := ctx.Args().First()
	if name == "" {
		return errors.New("conduit: missing `name\" argument")
	}

	ext := ctx.String("ext")
	if ext != "sql" && ext != "go" {
		return fmt.Errorf("conduit: unsupported extension \"%s\", expected \"sql\" or \"go\"", ext)
	}

	ver := time.Now().UnixMilli()
	filename := version.MigrationFilename(ver, name, ext)
	path := filepath.Join(dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"conduit: unable to create migration file %s: %w",
			path,
			err,
		)
	}
	defer f.Close()

	var tpl *template.Template
	switch ext {
	case "go":
		tpl = internaltpl.GoMigrationTemplate
	case "sql":
		tpl = internaltpl.SQLMigrationTemplate
	default:
		return errors.New("conduit: unsupported extension")
	}

	hasCustomRegistry := exists(filepath.Join(dir, "registry.go"))
	if err := tpl.Execute(f, struct {
		HasCustomRegistry bool
		Ext               string
		Name              string
		Version           int64
	}{hasCustomRegistry, ext, name, ver}); err != nil {
		return fmt.Errorf("conduit: unable to write template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf(
			"conduit: failed to write a migration file %s: %w",
			path,
			err,
		)
	}

	return nil
}

// exists check if a FS entry exists at path.
func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}
