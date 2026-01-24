package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/command/migrationctx"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/version"
	"go.inout.gg/conduit/pkg/pgdiff"
)

func diff(ctx context.Context, cmd *cli.Command, fs afero.Fs) error {
	migrationDir, err := migrationctx.Dir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}

	if !exists(fs, migrationDir) {
		return errors.New("migrations directory does not exist, try to initialise it first")
	}

	name := cmd.Args().First()
	if name == "" {
		return errors.New("missing `name` argument")
	}

	schemaPath := cmd.String("schema")

	connStr := cmd.String(flagname.DatabaseURL)
	image := cmd.String("image")

	var poolConfig *pgxpool.Config

	if connStr != "" {
		poolConfig, err = pgxpool.ParseConfig(connStr)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
	} else {
		var cleanup func(context.Context) error

		poolConfig, cleanup, err = pgdiff.StartPostgresContainer(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to start postgres container: %w", err)
		}

		defer func() {
			_ = cleanup(ctx)
		}()
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, poolConfig, migrationDir, schemaPath)
	if err != nil {
		return fmt.Errorf("failed to generate diff plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		//nolint:forbidigo
		fmt.Println("No schema changes detected.")
		return nil
	}

	ver := version.NewVersion()
	filename := version.MigrationFilename(ver, name, "sql")
	path := filepath.Join(migrationDir, filename)

	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create migration file %s: %w", path, err)
	}
	defer f.Close()

	var upStmts strings.Builder
	for i, stmt := range plan.Statements {
		upStmts.WriteString(stmt.ToSQL())

		if !strings.HasSuffix(stmt.ToSQL(), ";") {
			upStmts.WriteString(";")
		}

		if i < len(plan.Statements)-1 {
			upStmts.WriteString("\n\n")
		}
	}

	if err := internaltpl.DiffMigrationTemplate.Execute(f, struct {
		Version      version.Version
		Name         string
		SchemaFile   string
		UpStatements string
	}{
		Version:      ver,
		Name:         name,
		SchemaFile:   schemaPath,
		UpStatements: upStmts.String(),
	}); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to write migration file %s: %w", path, err)
	}

	//nolint:forbidigo
	fmt.Printf("Created migration: %s\n", path)

	// Print hazards if any
	for _, stmt := range plan.Statements {
		for _, hazard := range stmt.Hazards {
			//nolint:forbidigo
			fmt.Printf("Warning [%s]: %s\n", hazard.Type, hazard.Message)
		}
	}

	return nil
}
