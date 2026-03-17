// Package pgdiff compares PostgreSQL schemas and generates migration plans.
package pgdiff

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/afero"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/schema"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"

	"go.inout.gg/conduit/internal/migrationfile"
	"go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/sqlsplit"
)

// Plan holds the generated migration plan and the target schema hash.
type Plan struct {
	SourceSchemaHash string
	TargetSchemaHash string
	Statements       []schemadiff.Statement
}

// GeneratePlan compares the source schema (from migrationsDir) against the
// target schema (in schemaPath) and returns a plan with the required DDL
// statements and schema hashes.
func GeneratePlan(
	ctx context.Context,
	fs afero.Fs,
	connConfig *pgx.ConnConfig,
	migrationsDir, schemaPath string,
	excludeSchemas []string,
) (Plan, error) {
	var result Plan

	sourceMigrations, err := migrationfile.ReadMigrationsFromDir(fs, migrationsDir)
	if err != nil {
		return result, fmt.Errorf("failed to read migrations: %w", err)
	}

	var sourceStmts []sqlsplit.Stmt
	for _, m := range sourceMigrations {
		sourceStmts = append(sourceStmts, m.Stmts...)
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

	// Apply conduit's internal schema (e.g. conduit_migrations table) to the
	// target database so the schema hash includes it.
	if err := exec(ctx, targetDb.ConnPool, string(migrations.Schema)); err != nil {
		return result, fmt.Errorf("failed to execute conduit internal schema: %w", err)
	}

	planOpts := []schemadiff.PlanOpt{
		schemadiff.WithTempDbFactory(factory),
		schemadiff.WithGetSchemaOpts(targetDb.ExcludeMetadataOptions...),
	}
	schemaOpts := targetDb.ExcludeMetadataOptions

	if len(excludeSchemas) > 0 {
		planOpts = append(planOpts, schemadiff.WithExcludeSchemas(excludeSchemas...))
		schemaOpts = append(schemaOpts, schema.WithExcludeSchemas(excludeSchemas...))
	}

	// Include conduit's internal schema in the source DDL so it matches the
	// target and cancels out in the diff — only user schema changes remain.
	internalStmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return result, fmt.Errorf("failed to parse conduit internal schema: %w", err)
	}

	sourceDDL := append(
		sliceutil.Map(
			sliceutil.Filter(internalStmts, func(s sqlsplit.Stmt) bool {
				return s.Type == sqlsplit.StmtTypeQuery
			}),
			func(s sqlsplit.Stmt) string { return s.Content },
		),
		sliceutil.Map(sourceStmts, func(stmt sqlsplit.Stmt) string { return stmt.Content })...,
	)

	plan, err := schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource(sourceDDL),
		schemadiff.DBSchemaSource(targetDb.ConnPool),
		planOpts...,
	)
	if err != nil {
		return result, fmt.Errorf("failed to generate plan: %w", err)
	}

	hash, err := schema.GetSchemaHash(ctx, targetDb.ConnPool, schemaOpts...)
	if err != nil {
		return result, fmt.Errorf("failed to generate target schema hash: %w", err)
	}

	result.Statements = plan.Statements
	result.SourceSchemaHash = plan.CurrentSchemaHash
	result.TargetSchemaHash = hash

	return result, nil
}

// GenerateSchemaHashChain applies migrations incrementally to a temporary
// database and returns the cumulative schema hash after each migration group.
func GenerateSchemaHashChain(
	ctx context.Context,
	connConfig *pgx.ConnConfig,
	migrationGroups [][]sqlsplit.Stmt,
	excludeSchemas []string,
) ([]string, error) {
	internalStmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conduit internal schema: %w", err)
	}

	factory, err := newTempDbFactory(ctx, connConfig)
	if err != nil {
		return nil, err
	}
	defer factory.Close()

	db, err := factory.Create(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp db: %w", err)
	}
	defer db.Close(ctx)

	for _, stmt := range internalStmts {
		if stmt.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		if _, err := db.ConnPool.ExecContext(ctx, stmt.Content); err != nil {
			return nil, fmt.Errorf("failed to execute conduit internal schema: %w", err)
		}
	}

	schemaOpts := db.ExcludeMetadataOptions
	if len(excludeSchemas) > 0 {
		schemaOpts = append(schemaOpts, schema.WithExcludeSchemas(excludeSchemas...))
	}

	hashes := make([]string, 0, len(migrationGroups))

	for _, group := range migrationGroups {
		for _, stmt := range group {
			if stmt.Type != sqlsplit.StmtTypeQuery {
				continue
			}

			if _, err := db.ConnPool.ExecContext(ctx, stmt.Content); err != nil {
				return nil, fmt.Errorf("failed to execute statement: %w", err)
			}
		}

		hash, err := schema.GetSchemaHash(ctx, db.ConnPool, schemaOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema hash: %w", err)
		}

		hashes = append(hashes, hash)
	}

	return hashes, nil
}

// DumpSchema extracts the schema of a live Postgres database as DDL statements.
func DumpSchema(
	ctx context.Context,
	connConfig *pgx.ConnConfig,
	excludeSchemas []string,
) ([]schemadiff.Statement, error) {
	remoteDB := stdlib.OpenDB(*connConfig)
	defer remoteDB.Close()

	factory, err := newTempDbFactory(ctx, connConfig)
	if err != nil {
		return nil, err
	}
	defer factory.Close()

	planOpts := []schemadiff.PlanOpt{
		schemadiff.WithTempDbFactory(factory),
		schemadiff.WithDoNotValidatePlan(),
		schemadiff.WithNoConcurrentIndexOps(),
	}
	if len(excludeSchemas) > 0 {
		planOpts = append(planOpts, schemadiff.WithExcludeSchemas(excludeSchemas...))
	}

	// Use conduit's internal schema as the DDL baseline so that conduit-managed
	// tables (e.g. conduit_migrations) cancel out in the diff against the remote DB.
	internalStmts, err := sqlsplit.Split(migrations.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conduit internal schema: %w", err)
	}

	plan, err := schemadiff.Generate(
		ctx,
		schemadiff.DDLSchemaSource(sliceutil.Map(
			sliceutil.Filter(internalStmts, func(s sqlsplit.Stmt) bool {
				return s.Type == sqlsplit.StmtTypeQuery
			}),
			func(s sqlsplit.Stmt) string { return s.Content },
		)),
		schemadiff.DBSchemaSource(remoteDB),
		planOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dump schema: %w", err)
	}

	// Filter out any remaining conduit internal statements (e.g. when the
	// remote DB doesn't have conduit tables yet).
	return sliceutil.Filter(plan.Statements, func(s schemadiff.Statement) bool {
		return !strings.Contains(s.DDL, "conduit_migrations")
	}), nil
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

func exec(ctx context.Context, db *sql.DB, sql string) error {
	stmts, err := sqlsplit.Split([]byte(sql))
	if err != nil {
		return fmt.Errorf("failed to parse SQL: %w", err)
	}

	for _, stmt := range stmts {
		if stmt.Type != sqlsplit.StmtTypeQuery {
			continue
		}

		if _, err := db.ExecContext(ctx, stmt.Content); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}
