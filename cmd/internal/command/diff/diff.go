package diff

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/config"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/timegenerator"
)

const schemaFlag = "schema"

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator, bi buildinfo.BuildInfo, cfg *config.Config) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "diff",
		Usage: "create a migration from schema diff using pg-schema-diff",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:    schemaFlag,
				Usage:   "path to the target schema SQL file",
				Sources: cli.EnvVars("CONDUIT_SCHEMA"),
			},
			cmdutil.DatabaseURLFlag(),
			cmdutil.MigrationsDirFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return errors.New("missing `name` argument")
			}

			schemaPath, _ := config.FilePath(cfg.Migrations.Schema)

			schema := cmdutil.StringOr(cmd, schemaFlag, schemaPath)
			if schema == "" {
				return errors.New("missing `--schema` flag")
			}

			dirPath, _ := config.FilePath(cfg.Migrations.Dir)
			migrationsDir := cmdutil.StringOr(cmd, cmdutil.MigrationsDir, dirPath)
			dbURL := cmdutil.StringOr(cmd, cmdutil.DatabaseURL, cfg.Database.URL)

			args := conduitcli.DiffArgs{
				Dir:         filepath.Clean(migrationsDir),
				Name:        name,
				SchemaPath:  schema,
				DatabaseURL: dbURL,
			}

			return conduitcli.Diff(ctx, fs, timeGen, bi, args)
		},
	}
}
