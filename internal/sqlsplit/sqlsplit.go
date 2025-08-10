package sqlsplit

import (
	"fmt"

	pgquery "github.com/pganalyze/pg_query_go/v6"
)

func Split(sql string) ([]string, error) {
	stmts, err := pgquery.SplitWithParser(sql, true)
	if err != nil {
		return nil, fmt.Errorf("failed to split SQL: %w", err)
	}

	return stmts, nil
}
