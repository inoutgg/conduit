package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/command/migrationctx"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/pkg/version"
)

type EmptyArgs struct {
	Name        string
	Ext         string
	PackageName string
}

func empty(ctx context.Context, fs afero.Fs, args EmptyArgs) error {
	dir, err := migrationctx.Dir(ctx)
	if err != nil {
		return fmt.Errorf("conduit: failed to get migration directory: %w", err)
	}

	// Ensure migration dir exists.
	if !exists(fs, dir) {
		return errors.New("conduit: migrations directory does not exist, try to initialise it first")
	}

	ver := version.NewVersion()

	switch args.Ext {
	case "sql":
		if err := createSQLMigration(fs, dir, ver, args); err != nil {
			return err
		}
	case "go":
		if err := createGoMigration(fs, dir, ver, args); err != nil {
			return err
		}
	}

	return nil
}

func createSQLMigration(fs afero.Fs, dir string, ver version.Version, args EmptyArgs) error {
	tplData := struct {
		Version version.Version
		Name    string
	}{ver, args.Name}

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

	return writeTemplate(fs, path, internaltpl.GoMigrationTemplate, struct {
		Version           version.Version
		Ext               string
		Name              string
		Package           string
		HasCustomRegistry bool
	}{ver, args.Ext, args.Name, args.PackageName, hasCustomRegistry})
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

// exists check if a FS entry exists at path.
func exists(afs afero.Fs, path string) bool {
	_, err := afs.Stat(path)
	return !errors.Is(err, afero.ErrFileNotFound)
}
