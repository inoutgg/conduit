package conduitregistry

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/sqlsplit"
	"go.inout.gg/conduit/pkg/version"
)

// parseMigrationsFromFS scans the fsys for SQL migration scripts and returns.
func parseSQLMigrationsFromFS(fsys fs.FS, root string) ([]*Migration, error) {
	migrations := make([]*Migration, 0)

	err := fs.WalkDir(fsys, root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Matching the root directory, skip it.
		if path == root {
			return nil
		}

		migration, err := parseSQLMigration(fsys, path)
		if err != nil {
			return err
		}

		migrations = append(migrations, migration)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("conduit: error occurred while parsing migrations directory: %w", err)
	}

	return migrations, nil
}

// parseSQLMigration reads an SQL file from fsys by path and parses it
// into a migration.
func parseSQLMigration(fsys fs.FS, path string) (*Migration, error) {
	filename := filepath.Base(path)

	info, err := version.ParseMigrationFilename(filename)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to parse migration filename: %w", err)
	}

	sql, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to read migration file: %w", err)
	}

	up, down, err := sqlsplit.SplitMigration(string(sql))
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to split SQL statements: %w", err)
	}

	migration := Migration{
		version: info.Version,
		name:    info.Name,
		up:      emptyMigrateFunc,
		down:    emptyMigrateFunc,
	}

	migration.up = sqlMigrateFunc(up)

	// Down migration can be empty.
	if len(down) > 0 {
		migration.down = sqlMigrateFunc(down)
	}

	return &migration, nil
}

func sqlMigrateFunc(stmts []sqlsplit.Stmt) *migrateFunc {
	disableTx := slices.ContainsFunc(stmts, func(stmt sqlsplit.Stmt) bool {
		return stmt.Type == sqlsplit.StmtTypeDisableTx
	})

	sqlStmts := sliceutil.Filter(stmts, func(stmt sqlsplit.Stmt) bool {
		return stmt.Type == sqlsplit.StmtTypeQuery
	})

	useTx := !disableTx
	migration := &migrateFunc{useTx: useTx, fn: nil, fnx: nil}

	if useTx {
		migration.fnx = func(ctx context.Context, tx pgx.Tx) error {
			for _, stmt := range sqlStmts {
				if _, err := tx.Exec(ctx, stmt.Content); err != nil {
					return fmt.Errorf("conduit: failed to execute migration script: %w", err)
				}
			}

			return nil
		}
	} else {
		migration.fn = func(ctx context.Context, conn *pgx.Conn) error {
			for _, stmt := range sqlStmts {
				_, err := conn.Exec(ctx, stmt.Content)
				if err != nil {
					return fmt.Errorf("conduit: failed to execute migration script: %w", err)
				}
			}

			return nil
		}
	}

	return migration
}
