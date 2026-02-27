package conduitcli

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit/pkg/pgdiff"
)

type DumpArgs struct {
	DatabaseURL string
}

func Dump(ctx context.Context, w io.Writer, args DumpArgs) error {
	connConfig, err := pgx.ParseConfig(args.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	stmts, err := pgdiff.DumpSchema(ctx, connConfig)
	if err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	var sb bytes.Buffer

	for i, stmt := range stmts {
		if i > 0 {
			sb.WriteString("\n")
		}

		sb.WriteString(stmt.ToSQL())
		sb.WriteString("\n")
	}

	if _, err := w.Write(sb.Bytes()); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}
