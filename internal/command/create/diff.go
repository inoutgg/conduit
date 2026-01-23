package create

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/command/migrationctx"
	"go.inout.gg/conduit/internal/sqlsplit"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/version"
)

func diff(ctx context.Context, cmd *cli.Command) error {
	migrationDir, err := migrationctx.Dir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}

	if !exists(migrationDir) {
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

		poolConfig, cleanup, err = startEphemeralPostgres(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to start postgres container: %w", err)
		}

		defer func() {
			_ = cleanup(ctx)
		}()
	}

	plan, err := generateDiffPlan(ctx, poolConfig, migrationDir, schemaPath)
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

	f, err := os.Create(path)
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

// generateDiffPlan generates a migration plan by comparing the source schema
// from sourceMigrationDir against the target schema in targetSchemaFile.
//
//nolint:nonamedreturns
func generateDiffPlan(
	ctx context.Context,
	poolConfig *pgxpool.Config,
	sourceMigrationDir, targetSchemaFile string,
) (plan schemadiff.Plan, err error) {
	sourceStmts, err := extractStmtsFromMigrationsDir(sourceMigrationDir)
	if err != nil {
		return plan, fmt.Errorf("failed to extract statements from migrations: %w", err)
	}

	targetSchema, err := os.ReadFile(targetSchemaFile)
	if err != nil {
		return plan, fmt.Errorf("failed to read schema file %s: %w", targetSchemaFile, err)
	}

	targetStmts, _, err := sqlsplit.Split(string(targetSchema))
	if err != nil {
		return plan, fmt.Errorf("failed to parse target schema: %w", err)
	}

	tempDbFactory, err := tempdb.NewOnInstanceFactory(
		ctx,
		func(ctx context.Context, dbName string) (*sql.DB, error) {
			config := poolConfig.Copy()
			config.ConnConfig.Database = dbName

			p, err := pgxpool.NewWithConfig(ctx, config)
			if err != nil {
				return nil, fmt.Errorf("failed to create connection pool for %s: %w", dbName, err)
			}

			return stdlib.OpenDBFromPool(p), nil
		},
		tempdb.WithDbPrefix("conduit"),
	)
	if err != nil {
		return plan, fmt.Errorf("failed to create temp db factory: %w", err)
	}
	defer tempDbFactory.Close()

	plan, err = schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource(sourceStmts),
		schemadiff.DDLSchemaSource(targetStmts),
		schemadiff.WithTempDbFactory(tempDbFactory),
	)
	if err != nil {
		return plan, fmt.Errorf("failed to generate plan: %w", err)
	}

	return plan, nil
}

// extractStmtsFromMigrationsDir reads all SQL migration files from the directory
// and extracts the "up" statements.
func extractStmtsFromMigrationsDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var allStmts []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}

		stmts, _, err := sqlsplit.Split(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse SQL in %s: %w", name, err)
		}

		allStmts = append(allStmts, stmts...)
	}

	return allStmts, nil
}
