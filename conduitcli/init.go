package conduitcli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/migrations"
	conduittemplate "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/pkg/conduitsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/timegenerator"
	"go.inout.gg/conduit/pkg/version"
)

const configFilename = "conduit.yaml"

// InitArgs configures a project initialization operation.
type InitArgs struct {
	Dir            string
	MigrationsDir  string
	DatabaseURL    string
	ExcludeSchemas []string
}

// Init creates a new migrations directory with the initial conduit schema
// migration, generates a conduit.sum file with the baseline schema hash,
// and writes a default conduit.yaml config file.
func Init(ctx context.Context, fs afero.Fs, timeGen timegenerator.Generator, args InitArgs) error {
	migrationsPath := filepath.Join(args.Dir, args.MigrationsDir)

	if err := createMigrationDir(fs, migrationsPath); err != nil {
		return err
	}

	migrationsFs := afero.NewBasePathFs(fs, migrationsPath)
	ver := version.NewFromTime(timeGen.Now())

	if err := createInitialMigration(migrationsFs, ver); err != nil {
		return err
	}

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	stmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return fmt.Errorf("failed to parse initial schema: %w", err)
	}

	hash, err := pgdiff.GenerateSchemaHash(ctx, connConfig, stmts, args.ExcludeSchemas)
	if err != nil {
		return fmt.Errorf("failed to generate schema hash: %w", err)
	}

	if err := conduitsum.WriteFile(migrationsFs, hash); err != nil {
		return fmt.Errorf("failed to write conduit.sum: %w", err)
	}

	if err := writeConfigFile(fs, args.Dir, args.MigrationsDir, args.DatabaseURL); err != nil {
		return err
	}

	return nil
}

func writeConfigFile(fs afero.Fs, parentDir, migrationsName, databaseURL string) error {
	var buf bytes.Buffer

	data := struct {
		Dir         string
		DatabaseURL string
	}{
		Dir:         migrationsName,
		DatabaseURL: databaseURL,
	}

	if err := conduittemplate.ConduitYAMLTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render config template: %w", err)
	}

	configPath := filepath.Join(parentDir, configFilename)
	if err := afero.WriteFile(fs, configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func createMigrationDir(fs afero.Fs, dir string) error {
	err := fs.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create migrations directory at %s: %w", dir, err)
	}

	return nil
}

func createInitialMigration(fs afero.Fs, ver version.Version) error {
	filename := version.MigrationFilename(ver, "conduit_initial_schema", version.MigrationDirectionUp)

	if err := afero.WriteFile(fs, filename, migrations.Schema, 0o644); err != nil {
		return fmt.Errorf("failed to create initial migration file: %w", err)
	}

	return nil
}
