package conduitregistry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/internal/version"
)

var MigrationSep = "---- create above / drop below ----"

var disableTxPattern = regexp.MustCompile(`(?m)^---- disable-tx ----$`)

// parseMigrationsFromFS scans the fsys for SQL migration scripts and returns
// a list of migrations.
func parseSQLMigrationsFromFS(fsys fs.FS, root string) (migrations []*Migration, err error) {
	err = fs.WalkDir(fsys, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("conduit: error occurred while parsing migrations directory: %w", err)
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

	return migrations, err
}

// parseSQLMigration reads an SQL file from fsys by path and parses it
// into a migration.
func parseSQLMigration(fsys fs.FS, path string) (*Migration, error) {
	filename := filepath.Base(path)
	info, err := version.ParseMigrationFilename(filename)
	if err != nil {
		return nil, err
	}

	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	migration := Migration{version: info.Version, name: info.Name}
	up, down, ok := strings.Cut(string(content), MigrationSep)

	up = strings.TrimSpace(up)
	if up == "" {
		return nil, errors.New("conduit: empty migration script")
	}

	migration.up, err = sqlMigrateFunc(up)
	if err != nil {
		return nil, err
	}

	if ok {
		migration.down, err = sqlMigrateFunc(down)
		if err != nil {
			return nil, err
		}
	} else {
		migration.down = emptyMigrateFunc
	}

	return &migration, nil
}

func sqlMigrateFunc(sql string) (*migrateFunc, error) {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return nil, EmptyMigrationErr
	}

	inTx := disableTxPattern.Match([]byte(sql))
	sql = disableTxPattern.ReplaceAllLiteralString(sql, "")
	fn := &migrateFunc{inTx: inTx}

	if inTx {
		fn.fnx = func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		}
	} else {
		fn.fn = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, sql)
			return err
		}
	}

	return fn, nil
}
