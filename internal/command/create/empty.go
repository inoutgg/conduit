package create

import (
	"errors"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/spf13/afero"

	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/timegenerator"
	"go.inout.gg/conduit/pkg/version"
)

type EmptyArgs struct {
	Dir         string
	Name        string
	Ext         string
	PackageName string
}

func empty(fs afero.Fs, timeGen timegenerator.Generator, args EmptyArgs) error {
	// Ensure migration dir exists.
	if !exists(fs, args.Dir) {
		return errors.New("conduit: migrations directory does not exist, try to initialise it first")
	}

	ver := version.NewFromTime(timeGen.Now())

	switch args.Ext {
	case "sql":
		if err := createSQLMigration(fs, args.Dir, ver, args); err != nil {
			return err
		}
	case "go":
		if err := createGoMigration(fs, args.Dir, ver, args); err != nil {
			return err
		}
	}

	return nil
}

func createSQLMigration(fs afero.Fs, dir string, ver version.Version, args EmptyArgs) error {
	tplData := map[string]any{
		"Version": ver,
		"Name":    args.Name,
	}

	for _, pair := range []struct {
		tpl       *template.Template
		direction version.MigrationDirection
	}{
		{internaltpl.SQLUpMigrationTemplate, version.MigrationDirectionUp},
		{internaltpl.SQLDownMigrationTemplate, version.MigrationDirectionDown},
	} {
		filename, err := version.MigrationFilename(ver, args.Name, pair.direction, "sql")
		if err != nil {
			return fmt.Errorf("conduit: failed to generate migration filename: %w", err)
		}

		path := filepath.Join(dir, filename)

		if err := writeTemplate(fs, path, pair.tpl, tplData); err != nil {
			return err
		}
	}

	return nil
}

func createGoMigration(fs afero.Fs, dir string, ver version.Version, args EmptyArgs) error {
	filename, err := version.MigrationFilename(ver, args.Name, "", "go")
	if err != nil {
		return fmt.Errorf("conduit: failed to generate migration filename: %w", err)
	}

	path := filepath.Join(dir, filename)
	hasCustomRegistry := exists(fs, filepath.Join(dir, "registry.go"))

	return writeTemplate(fs, path, internaltpl.GoMigrationTemplate, map[string]any{
		"Version":           ver,
		"Ext":               args.Ext,
		"Name":              args.Name,
		"Package":           args.PackageName,
		"HasCustomRegistry": hasCustomRegistry,
	})
}

func writeTemplate(fs afero.Fs, path string, tpl *template.Template, data any) error {
	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("conduit: failed to create migration file %s: %w", path, err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("conduit: failed to write template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("conduit: failed to write migration file %s: %w", path, err)
	}

	return nil
}
