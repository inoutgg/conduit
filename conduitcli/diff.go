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

	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/conduitsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/timegenerator"
	"go.inout.gg/conduit/pkg/version"
)

//nolint:gochecknoglobals
var nonTxPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+CONCURRENTLY`),
	regexp.MustCompile(`(?i)DROP\s+INDEX\s+CONCURRENTLY`),
	regexp.MustCompile(`(?i)REINDEX\s+.*CONCURRENTLY`),
	regexp.MustCompile(`(?i)ALTER\s+TYPE\s+.*ADD\s+VALUE`),
}

type DiffArgs struct {
	Dir         string
	Name        string
	SchemaPath  string
	DatabaseURL string
}

func Diff(
	ctx context.Context,
	fs afero.Fs,
	timeGen timegenerator.Generator,
	bi buildinfo.BuildInfo,
	args DiffArgs,
) error {
	if !exists(fs, args.Dir) {
		return errors.New("migrations directory does not exist, try to initialise it first")
	}

	migrationsFs := afero.NewBasePathFs(fs, args.Dir)

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, connConfig, args.Dir, args.SchemaPath)
	if err != nil {
		return fmt.Errorf("failed to generate diff plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		return errors.New("no schema changes detected")
	}

	if expectedHash, err := conduitsum.ReadFile(migrationsFs); err == nil {
		if plan.SourceSchemaHash != expectedHash {
			return fmt.Errorf(
				"source schema drift detected: expected hash %s (from conduit.sum), got %s",
				expectedHash,
				plan.SourceSchemaHash,
			)
		}
	}

	v := version.NewFromTime(timeGen.Now())
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

		filename := version.MigrationFilename(v, name, version.MigrationDirectionUp)

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

	if err := conduitsum.WriteFile(migrationsFs, plan.TargetSchemaHash); err != nil {
		return fmt.Errorf("conduit: failed to write conduit.sum: %w", err)
	}

	return nil
}

type migration struct {
	stmts   []schemadiff.Statement
	isNonTx bool
}

// splitMigrations splits statements into contiguous groups based on whether
// they require non-transactional execution.
func splitMigrations(stmts []schemadiff.Statement) []migration {
	if len(stmts) == 0 {
		return nil
	}

	var migrations []migration

	//nolint:exhaustruct
	current := migration{isNonTx: isNonTxStmt(stmts[0].DDL)}

	for _, stmt := range stmts {
		isNonTx := isNonTxStmt(stmt.DDL)
		if isNonTx != current.isNonTx {
			migrations = append(migrations, current)

			//nolint:exhaustruct
			current = migration{isNonTx: isNonTx}
		}

		current.stmts = append(current.stmts, stmt)
	}

	migrations = append(migrations, current)

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
		return fmt.Errorf("conduit: failed to create migration file %s: %w", path, err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("conduit: failed to write template: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("conduit: failed to write migration file %s: %w", path, err)
	}

	return nil
}

func exists(afs afero.Fs, path string) bool {
	_, err := afs.Stat(path)
	return !errors.Is(err, afero.ErrFileNotFound)
}
