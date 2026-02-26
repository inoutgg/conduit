package pgdiff

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/afero"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/schema"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"

	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/version"
)

// Plan holds the generated migration plan and the target schema hash.
type Plan struct {
	SourceSchemaHash string
	TargetSchemaHash string
	Statements       []schemadiff.Statement
}

// GeneratePlan generates a migration plan by comparing the source schema
// from migrationsDir against the target schema in schemaPath. It also
// computes the target schema hash without creating an additional temp database.
func GeneratePlan(
	ctx context.Context,
	fs afero.Fs,
	connConfig *pgx.ConnConfig,
	migrationsDir, schemaPath string,
) (Plan, error) {
	var result Plan

	sourceStmts, err := readStmtsFromMigrationsDir(fs, migrationsDir)
	if err != nil {
		return result, fmt.Errorf("failed to read migrations: %w", err)
	}

	targetStmts, err := readStmtsFromFile(fs, schemaPath)
	if err != nil {
		return result, fmt.Errorf("failed to read schema file: %w", err)
	}

	factory, err := newTempDbFactory(ctx, connConfig)
	if err != nil {
		return result, err
	}
	defer factory.Close()

	targetDb, err := factory.Create(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to create target temp db: %w", err)
	}
	defer targetDb.Close(ctx)

	for _, stmt := range targetStmts {
		if stmt.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		if _, err := targetDb.ConnPool.ExecContext(ctx, stmt.Content); err != nil {
			return result, fmt.Errorf("failed to execute target schema statement: %w", err)
		}
	}

	plan, err := schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource(
			sliceutil.Map(sourceStmts, func(stmt sqlsplit.Stmt) string { return stmt.Content }),
		),
		schemadiff.DBSchemaSource(targetDb.ConnPool),
		schemadiff.WithTempDbFactory(factory),
		schemadiff.WithGetSchemaOpts(targetDb.ExcludeMetadataOptions...),
	)
	if err != nil {
		return result, fmt.Errorf("failed to generate plan: %w", err)
	}

	hash, err := schema.GetSchemaHash(ctx, targetDb.ConnPool, targetDb.ExcludeMetadataOptions...)
	if err != nil {
		return result, fmt.Errorf("failed to generate target schema hash: %w", err)
	}

	result.Statements = plan.Statements
	result.SourceSchemaHash = plan.CurrentSchemaHash
	result.TargetSchemaHash = hash

	return result, nil
}

// GenerateSchemaHash creates a temp database, applies the given DDL statements,
// and returns the schema hash.
func GenerateSchemaHash(
	ctx context.Context,
	connConfig *pgx.ConnConfig,
	stmts []sqlsplit.Stmt,
) (string, error) {
	factory, err := newTempDbFactory(ctx, connConfig)
	if err != nil {
		return "", err
	}
	defer factory.Close()

	db, err := factory.Create(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create temp db: %w", err)
	}
	defer db.Close(ctx)

	for _, stmt := range stmts {
		if stmt.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		if _, err := db.ConnPool.ExecContext(ctx, stmt.Content); err != nil {
			return "", fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	hash, err := schema.GetSchemaHash(ctx, db.ConnPool, db.ExcludeMetadataOptions...)
	if err != nil {
		return "", fmt.Errorf("failed to get schema hash: %w", err)
	}

	return hash, nil
}

// DumpSchema extracts the schema of a live Postgres database as DDL statements.
func DumpSchema(
	ctx context.Context,
	connConfig *pgx.ConnConfig,
) ([]schemadiff.Statement, error) {
	remoteDB := stdlib.OpenDB(*connConfig)
	defer remoteDB.Close()

	factory, err := newTempDbFactory(ctx, connConfig)
	if err != nil {
		return nil, err
	}
	defer factory.Close()

	plan, err := schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource([]string{}),
		schemadiff.DBSchemaSource(remoteDB),
		schemadiff.WithTempDbFactory(factory),
		schemadiff.WithDoNotValidatePlan(),
		schemadiff.WithNoConcurrentIndexOps(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dump schema: %w", err)
	}

	return plan.Statements, nil
}

func newTempDbFactory(ctx context.Context, connConfig *pgx.ConnConfig) (tempdb.Factory, error) {
	factory, err := tempdb.NewOnInstanceFactory(
		ctx,
		func(_ context.Context, dbName string) (*sql.DB, error) {
			cc := connConfig.Copy()
			cc.Database = dbName

			return stdlib.OpenDB(*cc), nil
		},
		tempdb.WithDbPrefix("conduit"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp db factory: %w", err)
	}

	return factory, nil
}

func readStmtsFromMigrationsDir(fs afero.Fs, dir string) ([]sqlsplit.Stmt, error) {
	entries, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

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
		filename := m.Filename()
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
