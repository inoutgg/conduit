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
	filename := version.MigrationFilename(ver, args.Name, args.Ext)
	path := filepath.Join(dir, filename)

	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf(
			"conduit: failed to create migration file %s: %w",
			path,
			err,
		)
	}
	defer f.Close()

	var tpl *template.Template

	switch args.Ext {
	case "go":
		tpl = internaltpl.GoMigrationTemplate
	case "sql":
		tpl = internaltpl.SQLMigrationTemplate
	}

	hasCustomRegistry := exists(fs, filepath.Join(dir, "registry.go"))
	if err := tpl.Execute(f, struct {
		Version           version.Version
		Ext               string
		Name              string
		Package           string
		HasCustomRegistry bool
	}{ver, args.Ext, args.Name, args.PackageName, hasCustomRegistry}); err != nil {
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
func exists(afs afero.Fs, path string) bool {
	_, err := afs.Stat(path)
	return !errors.Is(err, afero.ErrFileNotFound)
}
