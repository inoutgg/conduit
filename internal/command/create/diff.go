package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/afero"

	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/internal/timegenerator"
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

	poolConfig, err := pgxpool.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, poolConfig, args.Dir, args.SchemaPath)
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
			upStmts.WriteString(fmt.Sprintf("-- [WARNING/%s]: %s\n", hazard.Type, hazard.Message))
		}

		upStmts.WriteString(stmt.ToSQL())

		if i < len(plan.Statements)-1 {
			upStmts.WriteString("\n\n")
		}
	}

	// Create up migration.
	upFilename, err := version.MigrationFilename(v, args.Name, version.MigrationDirectionUp, "sql")
	if err != nil {
		return fmt.Errorf("failed to generate migration filename: %w", err)
	}

	upPath := filepath.Join(args.Dir, upFilename)
	if err := writeTemplate(fs, upPath, internaltpl.SQLUpMigrationTemplate, map[string]any{
		"Version":    v,
		"Name":       args.Name,
		"SchemaPath": args.SchemaPath,
		"UpStmts":    upStmts.String(),
	}); err != nil {
		return err
	}

	return nil
}
