package create

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/cmdutil"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/version"
)

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "create",
		Usage: "create a new migration file",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:  "ext",
				Usage: "migration file extension (values: \"go\", \"sql\")",
				Value: "sql",
			},
			cmdutil.PackageNameFlag,
		},
		Action: create,
	}
}

func create(ctx context.Context, cmd *cli.Command) error {
	dir, err := cmdutil.MigrationDir(ctx)
	if err != nil {
		return fmt.Errorf("conduit: failed to get migration directory: %w", err)
	}

	packageName := cmd.String("package-name")

	// Ensure migration dir exists.
	if !exists(dir) {
		return errors.New("conduit: migrations directory does not exist, try to initialise it first")
	}

	name := cmd.Args().First()
	if name == "" {
		return errors.New("conduit: missing `name\" argument")
	}

	ext := cmd.String("ext")
	if ext != "sql" && ext != "go" {
		return fmt.Errorf("conduit: unsupported extension \"%s\", expected \"sql\" or \"go\"", ext)
	}

	ver := version.NewVersion()
	filename := version.MigrationFilename(ver, name, ext)
	path := filepath.Join(dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"conduit: failed to create migration file %s: %w",
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
	}

	hasCustomRegistry := exists(filepath.Join(dir, "registry.go"))
	if err := tpl.Execute(f, struct {
		Version           version.Version
		Ext               string
		Name              string
		Package           string
		HasCustomRegistry bool
	}{ver, ext, name, packageName, hasCustomRegistry}); err != nil {
		return fmt.Errorf("conduit: failed to write template: %w", err)
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
