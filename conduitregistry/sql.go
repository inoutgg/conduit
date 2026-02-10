package conduitregistry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/version"
)

const DisableTxDirective = "---- disable-tx ----"

// parseSQLMigrationsFromFS scans the fsys for SQL migration scripts and returns.
func parseSQLMigrationsFromFS(fs afero.Fs, root string) ([]*Migration, error) {
	migrations := make(map[string]*Migration)

	err := afero.Walk(fs, root, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			return nil
		}

		info, err := version.ParseMigrationFilename(filepath.Base(path))
		if err != nil {
			return fmt.Errorf("conduit: failed to parse migration filename: %w", err)
		}

		content, err := afero.ReadFile(fs, path)
		if err != nil {
			return fmt.Errorf("conduit: failed to read migration file: %w", err)
		}

		stmts, err := sqlsplit.Split(content)
		if err != nil {
			return fmt.Errorf("conduit: failed to split migration SQL: %w", err)
		}

		key := info.Version.String()

		m, ok := migrations[key]
		if !ok {
			m = &Migration{
				version: info.Version,
				name:    info.Name,
				up:      nil,
				down:    emptyMigrateFunc,
			}
			migrations[key] = m
		}

		switch info.Direction {
		case version.MigrationDirectionUp:
			if m.up != nil {
				return fmt.Errorf(
					"conduit: duplicate up migration for version %s: %w",
					key,
					ErrUpExists,
				)
			}

			m.up = sqlMigrateFunc(stmts)

		case version.MigrationDirectionDown:
			if m.down != emptyMigrateFunc {
				return fmt.Errorf(
					"conduit: duplicate down migration for version %s: %w",
					key,
					ErrDownExists,
				)
			}

			m.down = sqlMigrateFunc(stmts)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("conduit: error occurred while parsing migrations directory: %w", err)
	}

	result := make([]*Migration, 0, len(migrations))
	for key, m := range migrations {
		if m.up == nil {
			return nil, fmt.Errorf("conduit: migration version %s has a down file but no up file", key)
		}

		result = append(result, m)
	}

	return result, nil
}

func sqlMigrateFunc(stmts []sqlsplit.Stmt) *migrateFunc {
	useTx := !slices.ContainsFunc(stmts, func(stmt sqlsplit.Stmt) bool {
		return stmt.Type == sqlsplit.StmtTypeComment &&
			strings.TrimSpace(stmt.Content) == DisableTxDirective
	})

	queryStmts := sliceutil.Filter(stmts, func(stmt sqlsplit.Stmt) bool {
		return stmt.Type == sqlsplit.StmtTypeQuery
	})

	migration := &migrateFunc{useTx: useTx, fn: nil, fnx: nil}

	if useTx {
		migration.fnx = func(ctx context.Context, tx pgx.Tx) error {
			for _, stmt := range queryStmts {
				if _, err := tx.Exec(ctx, stmt.Content); err != nil {
					return fmt.Errorf(
						"conduit: failed to execute migration script: %w\n\n%s",
						err,
						stmt.String(),
					)
				}
			}

			return nil
		}
	} else {
		migration.fn = func(ctx context.Context, conn *pgx.Conn) error {
			for _, stmt := range queryStmts {
				_, err := conn.Exec(ctx, stmt.Content)
				if err != nil {
					return fmt.Errorf("conduit: failed to execute migration script: %w\n\n%s", err, stmt.String())
				}
			}

			return nil
		}
	}

	return migration
}
