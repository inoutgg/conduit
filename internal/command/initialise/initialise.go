package initialise

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/common"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/version"
)

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			common.MigrationsDirFlag,

			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"ns"},
				Usage:   "if set, creates a custom registry with provided namespace",
			},

			//nolint:exhaustruct
			&cli.BoolFlag{
				Name:  "no-conduit-migrations",
				Usage: "if set, a migration file to create conduit's versioning table won't be included",
			},
		},

		Action: action,
	}
}

func action(ctx context.Context, cmd *cli.Command) error {
	dir, err := common.MigrationDir(ctx)
	if err != nil {
		//nolint:wrapcheck
		return err
	}

	if err := createMigrationDir(dir); err != nil {
		return err
	}

	ns := cmd.String("namespace")
	if ns != "" {
		if _, err := createRegistryFile(dir, ns); err != nil {
			return err
		}
	}

	if !cmd.Bool("no-conduit-migrations") {
		if _, err := createConduitMigrationFile(dir, ns); err != nil {
			return err
		}
	}

	return nil
}

// createMigrationDir creates a new migration file at the dir resolved from the current
// working directory.
func createMigrationDir(dir string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("conduit: failed to create migrations directory at %s: %w", dir, err)
	}

	return nil
}

// createConduitMigrationFile creates a new migration file with conduit's own migration file
// in the migrations directory.
func createConduitMigrationFile(dirpath string, namespace string) (string, error) {
	ver := version.NewVersion()
	filename := version.MigrationFilename(ver, "conduit_migration", "go")
	filepath := filepath.Join(dirpath, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("conduit: failed to create migrations file: %w", err)
	}
	defer f.Close()

	if err := internaltpl.ConduitMigrationTemplate.Execute(f, struct {
		Version           *version.Version
		HasCustomRegistry bool
	}{HasCustomRegistry: namespace != "", Version: ver}); err != nil {
		return "", fmt.Errorf("conduit: failed to write a template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return "", fmt.Errorf(
			"conduit: failed to write conduit migration file %s: %w",
			filename,
			err,
		)
	}

	return filename, nil
}

// createRegistryFile creates a custom migration registry in the migrations directory.
func createRegistryFile(dir string, ns string) (string, error) {
	filepath := filepath.Join(dir, "registry.go")

	f, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("conduit: failed to create registry file: %w", err)
	}

	defer f.Close()

	if err := internaltpl.RegistryTemplate.Execute(f, struct{ Namespace string }{Namespace: ns}); err != nil {
		return "", fmt.Errorf("conduit: failed to write a template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return "", fmt.Errorf(
			"conduit: failed to write migrations registry file %s: %w",
			filepath,
			err,
		)
	}

	return filepath, nil
}
