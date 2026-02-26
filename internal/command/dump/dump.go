package dump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/pkg/pgdiff"
)

//nolint:revive // ignore naming convention.
type DumpArgs struct {
	DatabaseURL string
}

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "dump",
		Usage: "dump schema DDL from a remote Postgres database",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:     flagname.DatabaseURL,
				Usage:    "database connection URL",
				Sources:  cli.EnvVars("CONDUIT_DATABASE_URL"),
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := DumpArgs{
				DatabaseURL: cmd.String(flagname.DatabaseURL),
			}

			return dump(ctx, args, os.Stdout)
		},
	}
}

func dump(ctx context.Context, args DumpArgs, w io.Writer) error {
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
