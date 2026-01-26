package create

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/internal/command/flagname"
)

func NewCommand(fs afero.Fs) *cli.Command {
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
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:    flagname.PackageName,
						Usage:   "package name",
						Value:   "migrations",
						Sources: cli.EnvVars("CONDUIT_PACKAGE_NAME"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().First()
					if name == "" {
						return errors.New("conduit: missing `name` argument")
					}

					ext := cmd.String("ext")
					if ext != "sql" && ext != "go" {
						return fmt.Errorf(
							"conduit: unsupported extension %q, expected \"sql\" or \"go\"",
							ext,
						)
					}

					args := EmptyArgs{
						Name:        name,
						Ext:         ext,
						PackageName: cmd.String(flagname.PackageName),
					}

					return empty(ctx, fs, args)
				},
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
						Name:    flagname.DatabaseURL,
						Usage:   "database connection URL",
						Sources: cli.EnvVars("CONDUIT_DATABASE_URL"),
					},
					//nolint:exhaustruct
					&cli.StringFlag{
						Name:  "image",
						Usage: "PostgreSQL Docker image for testcontainers (e.g., \"postgres:16-alpine\")",
						Value: "postgres:16-alpine",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().First()
					if name == "" {
						return errors.New("missing `name` argument")
					}

					args := DiffArgs{
						Name:        name,
						SchemaPath:  cmd.String("schema"),
						DatabaseURL: cmd.String(flagname.DatabaseURL),
						Image:       cmd.String("image"),
					}

					return diff(ctx, fs, args)
				},
			},
		},
	}
}
