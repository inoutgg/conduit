package sqlsplit

import (
	"fmt"
	"strings"
)

const UpDownSep = "---- create above / drop below ----"

// SplitMigration splits an SQL string into individual statements.
func SplitMigration(sql string) ([]Stmt, []Stmt, error) {
	stmts := strings.Split(sql, UpDownSep)

	if len(stmts) == 1 {
		up, err := split(sql)
		if err != nil {
			return nil, nil, err
		}

		return up, nil, nil
	}

	if len(stmts) != 2 {
		return nil, nil, fmt.Errorf("conduit: invalid SQL split, expected 2 parts, got %d", len(stmts))
	}

	up, err := split(stmts[0])
	if err != nil {
		return nil, nil, err
	}

	down, err := split(stmts[1])
	if err != nil {
		return nil, nil, err
	}

	return up, down, nil
}

func split(s string) ([]Stmt, error) {
	stmts, err := Split(s)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to split SQL: %w", err)
	}

	return stmts, nil
}
