package create

import (
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/cmdutil"
)

func NewCommand() *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "create",
		Usage: "create a new migration file",
		Commands: []*cli.Command{
			//nolint:exhaustruct
			{
				Name:  "empty",
				Usage: "create an empty migration file",
				Flags: []cli.Flag{
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:  "ext",
						Usage: "migration file extension (values: \"go\", \"sql\")",
						Value: "sql",
					},
					cmdutil.PackageNameFlag,
				},
				Action: empty,
			},
			//nolint:exhaustruct
			{
				Name:  "diff",
				Usage: "create a migration from schema diff using pg-schema-diff",
				Flags: []cli.Flag{
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:     "schema",
						Usage:    "path to the target schema SQL file",
						Required: true,
					},
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:  "database-url",
						Usage: "database URL for temp db factory (if not provided, uses testcontainers)",
					},
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:  "pg-version",
						Usage: "PostgreSQL version for testcontainers (e.g., \"16\", \"15\")",
						Value: "16",
					},
				},
				Action: diff,
			},
		},
	}
}
