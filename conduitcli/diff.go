package conduitcli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit"
	internaltpl "go.inout.gg/conduit/internal/conduittemplate"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/hashsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/timegenerator"
)

var (
	ErrMigrationsNotFound = errors.New("migrations directory not found")
	ErrNoChanges          = errors.New("no schema changes detected")
)

// DiffArgs configures a schema diff operation.
type DiffArgs struct {
	RootDir        string
	MigrationsDir  string
	Name           string
	SchemaPath     string
	DatabaseURL    string
	ExcludeSchemas []string
}

// DiffResultFile describes a single migration file created by a Diff operation.
type DiffResultFile struct {
	// Path is the path of the created migration file.
	Path string
}

// DiffResult holds the outcome of a Diff operation.
type DiffResult struct {
	// Files lists the migration files that were created.
	Files []DiffResultFile
}

// Diff compares the current migrations directory against a target schema file
// and generates a new migration file for each detected statement.
//
// When the schema is already in sync, Diff returns ErrNoChanges.
func Diff(
	ctx context.Context,
	fs afero.Fs,
	timeGen timegenerator.Generator,
	bi conduitbuildinfo.BuildInfo,
	store hashsum.Store,
	args DiffArgs,
) (*DiffResult, error) {
	if !exists(fs, args.MigrationsDir) {
		return nil, fmt.Errorf("%w: directory %q does not exist",
			ErrMigrationsNotFound, args.MigrationsDir)
	}

	migrationsFs := afero.NewBasePathFs(fs, args.MigrationsDir)

	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, connConfig, args.MigrationsDir, args.SchemaPath, args.ExcludeSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		return nil, ErrNoChanges
	}

	if ok, actual, err := store.Compare(args.RootDir, []byte(plan.SourceSchemaHash)); err == nil {
		if !ok {
			return nil, fmt.Errorf(
				"%w: expected hash %s, got %s",
				conduit.ErrSchemaDrift,
				actual,
				plan.SourceSchemaHash,
			)
		}
	}

	v := conduitversion.NewFromTime(timeGen.Now())

	var files []DiffResultFile

	for i, stmt := range plan.Statements {
		name := args.Name
		if len(plan.Statements) > 1 {
			name = fmt.Sprintf("%s_%d", args.Name, i+1)
		}

		var upStmts strings.Builder

		fmt.Fprintf(&upStmts, "SET statement_timeout = '%dms';\n", stmt.Timeout.Milliseconds())
		fmt.Fprintf(&upStmts, "SET lock_timeout = '%dms';\n", stmt.LockTimeout.Milliseconds())
		fmt.Fprintln(&upStmts)

		for _, hazard := range stmt.Hazards {
			fmt.Fprintf(&upStmts, "---- hazard: %s // %s ----\n", hazard.Type, hazard.Message)
		}

		upStmts.WriteString(stmt.ToSQL())

		filename := conduitversion.MigrationFilename(v, name, conduitversion.MigrationDirectionUp)

		if err := writeMigration(
			migrationsFs,
			filename,
			internaltpl.SQLUpMigrationTemplate,
			map[string]any{
				"SchemaPath":     args.SchemaPath,
				"ConduitVersion": bi.Version(),
				"UpStmts":        upStmts.String(),
			},
		); err != nil {
			return nil, err
		}

		files = append(files, DiffResultFile{
			Path: filepath.Join(args.MigrationsDir, filename),
		})
	}

	if err := store.Save(args.RootDir, []byte(plan.TargetSchemaHash)); err != nil {
		return nil, fmt.Errorf("failed to write hash sum: %w", err)
	}

	return &DiffResult{Files: files}, nil
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
