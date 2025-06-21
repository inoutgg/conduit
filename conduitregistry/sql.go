package conduitregistry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/internal/sqlsplit"
	"go.inout.gg/conduit/internal/version"
)

var (
	UpDownSep        = "-- create above / drop below ----" //nolint:gochecknoglobals
	DisableTxPattern = "-- disable-tx ----"                //nolint:gochecknoglobals
)

// parseMigrationsFromFS scans the fsys for SQL migration scripts and returns
// a list of migrations.
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

	stmts := sqlsplit.NewLexer(sql).Lex()
	if len(stmts) == 0 {
		return nil, errors.New("conduit: empty migration script")
	}

	splitIdx := sliceutil.Until(stmts, func(stmt *sqlsplit.Stmt) bool {
		if stmt.Kind() != sqlsplit.Comment {
			return false
		}

		return strings.TrimSpace(stmt.Content()) == UpDownSep
	})

	migration := Migration{
		version: info.Version,
		name:    info.Name,
		up:      emptyMigrateFunc,
		down:    emptyMigrateFunc,
	}

	migration.up = sqlMigrateFunc(stmts[:splitIdx])

	// Down migration can be empty.
	if splitIdx < len(stmts)-1 {
		migration.down = sqlMigrateFunc(stmts[splitIdx+1:])
	}

	return &migration, nil
}

func sqlMigrateFunc(stmts []*sqlsplit.Stmt) *migrateFunc {
	inTx := sliceutil.Some(stmts, func(stmt *sqlsplit.Stmt) bool {
		if stmt.Kind() != sqlsplit.Comment {
			return false
		}

		return strings.TrimSpace(stmt.Content()) == DisableTxPattern
	})
	up := &migrateFunc{inTx: inTx, fn: nil, fnx: nil}

	if inTx {
		up.fnx = func(ctx context.Context, tx pgx.Tx) error {
			for _, s := range stmts {
				if s.Kind() != sqlsplit.Query {
					continue
				}

				_, err := tx.Exec(ctx, s.Content())
				if err != nil {
					return fmt.Errorf("conduit: failed to execute migration script: %w", err)
				}
			}

			return nil
		}
	} else {
		up.fn = func(ctx context.Context, conn *pgx.Conn) error {
			for _, s := range stmts {
				if s.Kind() != sqlsplit.Query {
					continue
				}

				_, err := conn.Exec(ctx, s.Content())
				if err != nil {
					return fmt.Errorf("conduit: failed to execute migration script: %w", err)
				}
			}

			return nil
		}
	}

	return up
}
