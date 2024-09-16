package conduitregistry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/internal/version"
)

var (
	MigrationSep = "---- create above / drop below ----"
)

var _ Migration = (*sqlMigration)(nil)

type sqlMigration struct {
	version int64
	name    string

	up   string
	down string
}

func (m *sqlMigration) Version() int64 { return m.version }
func (m *sqlMigration) Name() string   { return m.name }

func (m *sqlMigration) Up(ctx context.Context, tx pgx.Tx) error   { return nil }
func (m *sqlMigration) Down(ctx context.Context, tx pgx.Tx) error { return nil }

// parseMigrationsFromFS scans the fsys for SQL migration scripts and returns
// a list of migrations.
func parseSQLMigrationsFromFS(fsys fs.FS, root string) (migrations []*sqlMigration, err error) {
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
func parseSQLMigration(fsys fs.FS, path string) (*sqlMigration, error) {
	filename := filepath.Base(path)
	info, err := version.ParseMigrationFilename(filename)
	if err != nil {
		return nil, err
	}

	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	migration := sqlMigration{version: info.Version}
	up, down, ok := strings.Cut(string(content), MigrationSep)

	up = strings.TrimSpace(up)
	if up == "" {
		return nil, errors.New("conduit: empty migration script")
	}
	migration.up = up

	if ok {
		migration.down = strings.TrimSpace(down)
	}

	return &migration, nil
}
