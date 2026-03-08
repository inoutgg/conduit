package conduitcli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/conduittemplate"
	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/timegenerator"
)

// InitArgs configures an [Init] operation.
type InitArgs struct {
	RootDir        string
	ConfigName     string
	MigrationsDir  string
	DatabaseURL    string
	ExcludeSchemas []string
}

// InitResult holds the paths created by [Init].
type InitResult struct {
	MigrationsDirPath string
	MigrationPath     string
	ConfigPath        string
	SumPath           string
}

// Init scaffolds a new conduit project: migrations directory, initial schema
// migration, conduit.sum, and conduit.yaml config file.
func Init(
	ctx context.Context,
	fs afero.Fs,
	timeGen timegenerator.Generator,
	store hashsum.Store,
	args InitArgs,
) (*InitResult, error) {
	migrationsPath := filepath.Join(args.RootDir, args.MigrationsDir)
	if err := createMigrationDir(fs, migrationsPath); err != nil {
		return nil, err
	}

	migrationsFs := afero.NewBasePathFs(fs, migrationsPath)
	migrationFilename := conduitversion.MigrationFilename(
		conduitversion.NewFromTime(timeGen.Now()),
		"conduit_initial_schema",
		conduitversion.MigrationDirectionUp,
	)

	if err := createInitialMigration(migrationsFs, migrationFilename); err != nil {
		return nil, err
	}

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	stmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial schema: %w", err)
	}

	hash, err := pgdiff.GenerateSchemaHash(ctx, connConfig, stmts, args.ExcludeSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema hash: %w", err)
	}

	if err := store.Save(args.RootDir, []byte(hash)); err != nil {
		return nil, fmt.Errorf("failed to write conduit.sum: %w", err)
	}

	if err := writeConfigFile(fs, args.RootDir, args.ConfigName, ConfigArgs{
		MigrationsDir: args.MigrationsDir,
		DatabaseURL:   args.DatabaseURL,
	}); err != nil {
		return nil, err
	}

	return &InitResult{
		MigrationsDirPath: args.MigrationsDir,
		MigrationPath:     filepath.Join(args.MigrationsDir, migrationFilename),
		ConfigPath:        args.ConfigName,
		SumPath:           "conduit.sum",
	}, nil
}

type ConfigArgs struct {
	MigrationsDir string
	DatabaseURL   string
}

func writeConfigFile(fs afero.Fs, dir string, name string, args ConfigArgs) error {
	var buf bytes.Buffer
	if err := conduittemplate.ConduitYAMLTemplate.Execute(&buf, args); err != nil {
		return fmt.Errorf("failed to render config template: %w", err)
	}

	configPath := filepath.Join(dir, name)
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

func createInitialMigration(fs afero.Fs, filename string) error {
	if err := afero.WriteFile(fs, filename, migrations.Schema, 0o644); err != nil {
		return fmt.Errorf("failed to create initial migration file: %w", err)
	}

	return nil
}
