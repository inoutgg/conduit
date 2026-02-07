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

	tplData := struct {
		Version    version.Version
		Name       string
		SchemaPath string
		UpStmts    string
	}{
		Version:    ver,
		Name:       args.Name,
		SchemaPath: args.SchemaPath,
		UpStmts:    upStmts.String(),
	}

	// Create up migration.
	upFilename, err := version.MigrationFilename(ver, args.Name, version.MigrationDirectionUp, "sql")
	if err != nil {
		return fmt.Errorf("failed to generate migration filename: %w", err)
	}

	upPath := filepath.Join(migrationDir, upFilename)
	if err := writeTemplate(fs, upPath, internaltpl.SQLUpMigrationTemplate, tplData); err != nil {
		return err
	}

	// Create down migration.
	downFilename, err := version.MigrationFilename(ver, args.Name, version.MigrationDirectionDown, "sql")
	if err != nil {
		return fmt.Errorf("failed to generate migration filename: %w", err)
	}

	downPath := filepath.Join(migrationDir, downFilename)
	if err := writeTemplate(fs, downPath, internaltpl.SQLDownMigrationTemplate, tplData); err != nil {
		return err
	}

	//nolint:forbidigo
	fmt.Printf("Created migration: %s\n", upPath)

	// Print hazards if any
	for _, stmt := range plan.Statements {
		for _, hazard := range stmt.Hazards {
			//nolint:forbidigo
			fmt.Printf("Warning [%s]: %s\n", hazard.Type, hazard.Message)
		}
	}

	return nil
}
