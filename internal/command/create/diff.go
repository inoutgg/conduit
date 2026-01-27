package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/command/migrationctx"
	internaltpl "go.inout.gg/conduit/internal/template"
	"go.inout.gg/conduit/pkg/pgdiff"
	"go.inout.gg/conduit/pkg/version"
)

type DiffArgs struct {
	Name        string
	SchemaPath  string
	DatabaseURL string
	Image       string
}

func diff(ctx context.Context, fs afero.Fs, args DiffArgs) error {
	migrationDir, err := migrationctx.Dir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}

	if !exists(fs, migrationDir) {
		return errors.New("migrations directory does not exist, try to initialise it first")
	}

	var poolConfig *pgxpool.Config

	if args.DatabaseURL != "" {
		poolConfig, err = pgxpool.ParseConfig(args.DatabaseURL)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
	} else {
		var cleanup func(context.Context) error

		poolConfig, cleanup, err = pgdiff.StartPostgresContainer(ctx, args.Image)
		if err != nil {
			return fmt.Errorf("failed to start postgres container: %w", err)
		}

		defer func() {
			_ = cleanup(ctx)
		}()
	}

	plan, err := pgdiff.GeneratePlan(ctx, fs, poolConfig, migrationDir, args.SchemaPath)
	if err != nil {
		return fmt.Errorf("failed to generate diff plan: %w", err)
	}

	if len(plan.Statements) == 0 {
		//nolint:forbidigo
		fmt.Println("No schema changes detected.")
		return nil
	}

	ver := version.NewVersion()
	filename := version.MigrationFilename(ver, args.Name, "sql")
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

	if err := internaltpl.SQLMigrationTemplate.Execute(f, struct {
		Version    version.Version
		Name       string
		SchemaPath string
		UpStmts    string
	}{
		Version:    ver,
		Name:       args.Name,
		SchemaPath: args.SchemaPath,
		UpStmts:    upStmts.String(),
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
