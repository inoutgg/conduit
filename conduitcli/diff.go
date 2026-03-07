package conduitcli

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"
	schemadiff "github.com/stripe/pg-schema-diff/pkg/diff"

	"go.inout.gg/conduit"
	internaltpl "go.inout.gg/conduit/internal/conduittemplate"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/timegenerator"
)

var ErrMigrationsNotFound = errors.New("migrations directory not found")

//nolint:gochecknoglobals
var nonTxPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+CONCURRENTLY`),
	regexp.MustCompile(`(?i)DROP\s+INDEX\s+CONCURRENTLY`),
	regexp.MustCompile(`(?i)REINDEX\s+.*CONCURRENTLY`),
	regexp.MustCompile(`(?i)ALTER\s+TYPE\s+.*ADD\s+VALUE`),
}

// DiffArgs configures a schema diff operation.
type DiffArgs struct {
	RootDir        string
	MigrationsDir  string
	Name           string
	SchemaPath     string
	DatabaseURL    string
	ExcludeSchemas []string
}

// Diff compares the current migrations directory against a target schema file
// and generates new migration files for any detected differences.
//
// Statements that require non-transactional execution (e.g. CREATE INDEX
// CONCURRENTLY) are automatically split into separate migration files.
func Diff(
	ctx context.Context,
	fs afero.Fs,
	timeGen timegenerator.Generator,
	bi conduitbuildinfo.BuildInfo,
	store hashsum.Store,
	args DiffArgs,
) error {
	if !exists(fs, args.MigrationsDir) {
		return fmt.Errorf("%w: directory %q does not exist", ErrMigrationsNotFound, args.MigrationsDir)
	}

	migrationsFs := afero.NewBasePathFs(fs, args.MigrationsDir)

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, connConfig, args.MigrationsDir, args.SchemaPath, args.ExcludeSchemas)
	if err != nil {
		return fmt.Errorf("failed to generate diff plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		return errors.New("no schema changes detected")
	}

	if ok, actual, err := store.Compare(args.RootDir, []byte(plan.SourceSchemaHash)); err == nil {
		if !ok {
			return fmt.Errorf(
				"%w: expected hash %s, got %s",
				conduit.ErrSchemaDrift,
				actual,
				plan.SourceSchemaHash,
			)
		}
	}

	v := conduitversion.NewFromTime(timeGen.Now())
	migrations := splitMigrations(plan.Statements)

	for i, m := range migrations {
		name := args.Name
		if len(migrations) > 1 {
			name = fmt.Sprintf("%s_%d", args.Name, i+1)
		}

		// Compute the max timeout across all statements in this migration group.
		// statement_timeout is applied by PostgreSQL to each statement individually,
		// not to the migration as a whole.
		var stmtTimeout, lockTimeout time.Duration
		for _, stmt := range m.stmts {
			stmtTimeout = max(stmtTimeout, stmt.Timeout)
			lockTimeout = max(lockTimeout, stmt.LockTimeout)
		}

		var upStmts strings.Builder

		if stmtTimeout > 0 {
			fmt.Fprintf(&upStmts, "SET statement_timeout = '%dms';\n", stmtTimeout.Milliseconds())
		}

		if lockTimeout > 0 {
			fmt.Fprintf(&upStmts, "SET lock_timeout = '%dms';\n", lockTimeout.Milliseconds())
		}

		if stmtTimeout > 0 || lockTimeout > 0 {
			upStmts.WriteString("\n")
		}

		for j, stmt := range m.stmts {
			for _, hazard := range stmt.Hazards {
				fmt.Fprintf(&upStmts, "---- hazard: %s // %s ----\n", hazard.Type, hazard.Message)
			}

			upStmts.WriteString(stmt.ToSQL())

			if j < len(m.stmts)-1 {
				upStmts.WriteString("\n\n")
			}
		}

		filename := conduitversion.MigrationFilename(v, name, conduitversion.MigrationDirectionUp)

		if err := writeMigration(
			migrationsFs,
			filename,
			internaltpl.SQLUpMigrationTemplate,
			map[string]any{
				"SchemaPath":     args.SchemaPath,
				"ConduitVersion": bi.Version(),
				"UpStmts":        upStmts.String(),
				"DisableTx":      m.isNonTx,
			},
		); err != nil {
			return err
		}
	}

	if err := store.Save(args.RootDir, []byte(plan.TargetSchemaHash)); err != nil {
		return fmt.Errorf("failed to write hash sum: %w", err)
	}

	return nil
}

type migration struct {
	stmts   []schemadiff.Statement
	isNonTx bool
}

func splitMigrations(stmts []schemadiff.Statement) []migration {
	if len(stmts) == 0 {
		return nil
	}

	var (
		migrations []migration
		current    *migration
	)

	for _, stmt := range stmts {
		if isNonTxStmt(stmt.DDL) {
			if current != nil {
				migrations = append(migrations, *current)
				current = nil
			}

			migrations = append(migrations, migration{
				stmts:   []schemadiff.Statement{stmt},
				isNonTx: true,
			})

			continue
		}

		if current == nil {
			current = &migration{} //nolint:exhaustruct
		}

		current.stmts = append(current.stmts, stmt)
	}

	if current != nil {
		migrations = append(migrations, *current)
	}

	return migrations
}

func isNonTxStmt(ddl string) bool {
	for _, p := range nonTxPatterns {
		if p.MatchString(ddl) {
			return true
		}
	}

	return false
}

func writeMigration(fs afero.Fs, path string, tpl *template.Template, data any) error {
	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create migration file %s: %w", path, err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to render migration template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to write migration file %s: %w", path, err)
	}

	return nil
}

func exists(afs afero.Fs, path string) bool {
	_, err := afs.Stat(path)
	return !errors.Is(err, afero.ErrFileNotFound)
}
