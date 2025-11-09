package sqlsplit

import (
	"fmt"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
)

const UpDownSep = "---- create above / drop below ----"

// Split splits an SQL string into individual statements.
func Split(sql string) ([]string, []string, error) {
	parts := strings.Split(sql, UpDownSep)

	if len(parts) == 1 {
		up, err := split(sql)
		if err != nil {
			return nil, nil, err
		}

		return up, nil, nil
	}

	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("conduit: invalid SQL split, expected 2 parts, got %d", len(parts))
	}

	up, err := split(parts[0])
	if err != nil {
		return nil, nil, err
	}

	down, err := split(parts[1])
	if err != nil {
		return nil, nil, err
	}

	return up, down, nil
}

func split(s string) ([]string, error) {
	stmts, err := pgquery.SplitWithParser(s, true)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to split SQL: %w", err)
	}

	return stmts, nil
}
