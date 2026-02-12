package initialise

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/timegenerator"
	"go.inout.gg/conduit/pkg/version"
)

//nolint:revive // ignore naming convention.
type InitialiseArgs struct {
	Dir                 string
	Namespace           string
	PackageName         string
	NoConduitMigrations bool
}

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    flagname.MigrationsDir,
				Usage:   "directory with migration files",
				Value:   "./migrations",
				Sources: cli.EnvVars("CONDUIT_MIGRATION_DIR"),
			},
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    flagname.PackageName,
				Usage:   "package name",
				Value:   "migrations",
				Sources: cli.EnvVars("CONDUIT_PACKAGE_NAME"),
			},

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

		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := InitialiseArgs{
				Dir:                 filepath.Clean(cmd.String(flagname.MigrationsDir)),
				Namespace:           cmd.String("namespace"),
				PackageName:         cmd.String(flagname.PackageName),
				NoConduitMigrations: cmd.Bool("no-conduit-migrations"),
			}

			return initialise(fs, timeGen, args)
		},
	}
}

func initialise(fs afero.Fs, timeGen timegenerator.Generator, args InitialiseArgs) error {
	if err := createMigrationDir(fs, args.Dir); err != nil {
		return err
	}

	if args.Namespace != "" {
		if _, err := createRegistryFile(fs, args.Dir, args.Namespace); err != nil {
			return err
		}
	}

	if !args.NoConduitMigrations {
		if _, err := createConduitMigrationFile(
			fs,
			args.Dir,
			args.Namespace,
			args.PackageName,
			timeGen,
		); err != nil {
			return err
		}
	}

	return nil
}

// createMigrationDir creates a new migration file at the dir resolved from the current
// working directory.
func createMigrationDir(fs afero.Fs, dir string) error {
	err := fs.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("conduit: failed to create migrations directory at %s: %w", dir, err)
	}

	return nil
}

// createConduitMigrationFile creates a new migration file with conduit's own migration file
// in the migrations directory.
func createConduitMigrationFile(
	fs afero.Fs,
	dirpath string,
	namespace string,
	packageName string,
	timeGen timegenerator.Generator,
) (string, error) {
	ver := version.NewFromTime(timeGen.Now())

	filename, err := version.MigrationFilename(ver, "conduit_migration", "", "go")
	if err != nil {
		return "", fmt.Errorf("conduit: failed to generate migration filename: %w", err)
	}

	fpath := filepath.Join(dirpath, filename)

	f, err := fs.Create(fpath)
	if err != nil {
		return "", fmt.Errorf("conduit: failed to create migrations file: %w", err)
	}
	defer f.Close()

	if err := internaltpl.ConduitMigrationTemplate.Execute(f, struct {
		Version           version.Version
		Package           string
		HasCustomRegistry bool
	}{HasCustomRegistry: namespace != "", Version: ver, Package: packageName}); err != nil {
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
func createRegistryFile(fs afero.Fs, dir string, ns string) (string, error) {
	fpath := filepath.Join(dir, "registry.go")

	f, err := fs.Create(fpath)
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
			fpath,
			err,
		)
	}

	return fpath, nil
}
