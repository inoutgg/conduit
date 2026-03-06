package diff

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	yamlsrc "github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/timegenerator"
)

const schemaFlag = "schema"

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator, bi buildinfo.BuildInfo, src altsrc.Sourcer) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "diff",
		Usage: "create a migration from schema diff using pg-schema-diff",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:  schemaFlag,
				Usage: "path to the target schema SQL file",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_SCHEMA"),
					yamlsrc.YAML("migrations.schema", src),
				),
			},
			cmdutil.ExcludeSchemasFlag(src),
			cmdutil.DatabaseURLFlag(src),
			cmdutil.MigrationsDirFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return errors.New("missing required argument: <name>")
			}

			schema := cmd.String(schemaFlag)
			if schema == "" {
				return fmt.Errorf("missing required flag: --%s", schemaFlag)
			}

			args := conduitcli.DiffArgs{
				Dir:            filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				Name:           name,
				SchemaPath:     schema,
				DatabaseURL:    cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas: cmd.StringSlice(cmdutil.ExcludeSchemas),
			}

			return conduitcli.Diff(ctx, fs, timeGen, bi, args)
		},
	}
}
