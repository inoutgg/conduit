package sqlsplit

import (
	"errors"
	"fmt"
)

// SplitMigration splits an SQL string into up and down statements.
func SplitMigration(sql string) ([]Stmt, []Stmt, error) {
	stmts, err := Split(sql)
	if err != nil {
		return nil, nil, fmt.Errorf("conduit: failed to split SQL: %w", err)
	}

	sepIdx := -1

	for i, stmt := range stmts {
		if stmt.Type == StmtTypeUpDownSep {
			if sepIdx != -1 {
				return nil, nil, errors.New("conduit: multiple separators found")
			}

			sepIdx = i
		}
	}

	if sepIdx == -1 {
		return stmts, nil, nil
	}

	return stmts[:sepIdx], stmts[sepIdx+1:], nil
}
