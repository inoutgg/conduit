// Package migrationfile provides utilities for reading migration SQL files.
package migrationfile

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/sqlsplit"
)

// ReadStmtsFromDir reads all up-migration SQL files from dir, ordered by
// version, and returns the parsed statements.
func ReadStmtsFromDir(fs afero.Fs, dir string) ([]sqlsplit.Stmt, error) {
	entries, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	migrations := make([]conduitversion.ParsedMigrationFilename, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}

		m, err := conduitversion.ParseMigrationFilename(name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration filename %s: %w", name, err)
		}

		if m.Direction != conduitversion.MigrationDirectionUp {
			continue
		}

		migrations = append(migrations, m)
	}

	slices.SortFunc(migrations, func(a, b conduitversion.ParsedMigrationFilename) int {
		return a.Compare(b)
	})

	var allStmts []sqlsplit.Stmt

	for _, m := range migrations {
		filename := m.Filename()
		path := filepath.Join(dir, filename)

		stmts, err := readStmtsFromFile(fs, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		allStmts = append(allStmts, stmts...)
	}

	return allStmts, nil
}

func readStmtsFromFile(fs afero.Fs, path string) ([]sqlsplit.Stmt, error) {
	content, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	stmts, err := sqlsplit.Split(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	return stmts, nil
}
