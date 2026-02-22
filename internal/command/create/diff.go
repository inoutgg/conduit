package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/timegenerator"
	"go.inout.gg/conduit/pkg/conduitsum"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/version"
)

type DiffArgs struct {
	Dir         string
	Name        string
	SchemaPath  string
	DatabaseURL string
}

func diff(ctx context.Context, fs afero.Fs, timeGen timegenerator.Generator, args DiffArgs) error {
	if !exists(fs, args.Dir) {
		return errors.New("migrations directory does not exist, try to initialise it first")
	}

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

	v := version.NewFromTime(timeGen.Now())

	var upStmts strings.Builder

	for i, stmt := range plan.Statements {
		for _, hazard := range stmt.Hazards {
			fmt.Fprintf(&upStmts, "-- [WARNING/%s]: %s\n", hazard.Type, hazard.Message)
		}

		upStmts.WriteString(stmt.ToSQL())

		if i < len(plan.Statements)-1 {
			upStmts.WriteString("\n\n")
		}
	}

	// Create up migration.
	upFilename := version.MigrationFilename(v, args.Name, version.MigrationDirectionUp)

	upPath := filepath.Join(args.Dir, upFilename)
	if err := writeTemplate(fs, upPath, internaltpl.SQLUpMigrationTemplate, map[string]any{
		"Version":    v,
		"Name":       args.Name,
		"SchemaPath": args.SchemaPath,
		"UpStmts":    upStmts.String(),
	}); err != nil {
		return err
	}

	currentHash := plan.TargetSchemaHash

	// Read existing conduit.sum and append the new hash.
	sumPath := filepath.Join(args.Dir, conduitsum.Filename)

	hashes := make([]string, 0, 1)

	if existingData, err := afero.ReadFile(fs, sumPath); err == nil {
		hashes, err = conduitsum.Parse(existingData)
		if err != nil {
			return fmt.Errorf("failed to parse conduit.sum: %w", err)
		}
	}

	hashes = append(hashes, currentHash)

	if err := afero.WriteFile(fs, sumPath, conduitsum.Format(hashes), 0o644); err != nil {
		return fmt.Errorf("conduit: failed to write conduit.sum: %w", err)
	}

	return nil
}

func writeTemplate(fs afero.Fs, path string, tpl *template.Template, data any) error {
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
