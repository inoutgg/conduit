package pgdiff

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/afero"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"

	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/version"
)

// GeneratePlan generates a migration plan by comparing the source schema
// from migrationsDir against the target schema in schemaPath.
func GeneratePlan(
	ctx context.Context,
	fs afero.Fs,
	poolConfig *pgxpool.Config,
	migrationsDir, schemaPath string,
) (schemadiff.Plan, error) {
	sourceStmts, err := readStmtsFromMigrationsDir(fs, migrationsDir)
	if err != nil {
		return schemadiff.Plan{}, fmt.Errorf("failed to read migrations: %w", err)
	}

	targetStmts, err := readStmtsFromFile(fs, schemaPath)
	if err != nil {
		return schemadiff.Plan{}, fmt.Errorf("failed to read schema file: %w", err)
	}

	return generatePlan(ctx, poolConfig, sourceStmts, targetStmts)
}

func generatePlan(
	ctx context.Context,
	poolConfig *pgxpool.Config,
	sourceStmts, targetStmts []sqlsplit.Stmt,
) (schemadiff.Plan, error) {
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
		return schemadiff.Plan{}, fmt.Errorf("failed to create temp db factory: %w", err)
	}
	defer tempDbFactory.Close()

	plan, err := schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource(
			sliceutil.Map(sourceStmts, func(stmt sqlsplit.Stmt) string { return stmt.Content }),
		),
		schemadiff.DDLSchemaSource(
			sliceutil.Map(targetStmts, func(stmt sqlsplit.Stmt) string { return stmt.Content }),
		),
		schemadiff.WithTempDbFactory(tempDbFactory),
	)
	if err != nil {
		return schemadiff.Plan{}, fmt.Errorf("failed to generate plan: %w", err)
	}

	return plan, nil
}

func readStmtsFromMigrationsDir(fs afero.Fs, dir string) ([]sqlsplit.Stmt, error) {
	entries, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter and parse up migration files
	migrations := make([]version.ParsedMigrationFilename, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}

		m, err := version.ParseMigrationFilename(name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration filename %s: %w", name, err)
		}

		if m.Direction != version.MigrationDirectionUp {
			continue
		}

		migrations = append(migrations, m)
	}

	slices.SortFunc(migrations, func(a, b version.ParsedMigrationFilename) int {
		return a.Compare(b)
	})

	var allStmts []sqlsplit.Stmt

	for _, m := range migrations {
		filename, err := m.Filename()
		if err != nil {
			return nil, fmt.Errorf("failed to construct migration filename: %w", err)
		}

		path := filepath.Join(dir, filename)

		stmts, err := readStmtsFromFile(fs, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		allStmts = append(allStmts, stmts...)
	}

	return allStmts, nil
}

func readStmtsFromFile(fs afero.Fs, path string) ([]sqlsplit.Stmt, error) {
	content, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	stmts, err := sqlsplit.Split(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	return stmts, nil
}
