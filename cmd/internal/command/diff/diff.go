package diff

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	yamlsrc "github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
	"go.inout.gg/conduit/pkg/lockfile"
	"go.inout.gg/conduit/pkg/timegenerator"
)

const schemaFlag = "schema"

func NewCommand(
	fs afero.Fs,
	_ io.Writer,
	stderr io.Writer,
	timeGen timegenerator.Generator,
	bi conduitbuildinfo.BuildInfo,
	src altsrc.Sourcer,
) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:  "diff",
		Usage: "create a migration from schema diff using pg-schema-diff",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:     schemaFlag,
				Usage:    "path to the target schema SQL file",
				Required: true,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("CONDUIT_SCHEMA"),
					yamlsrc.YAML("migrations.schema", src),
				),
			},
			cmdutil.SkipSchemaDriftCheckFlag(src),
			cmdutil.ExcludeSchemasFlag(src),
			cmdutil.DatabaseURLFlag(src),
			cmdutil.MigrationsDirFlag(src),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return errors.New("missing required argument: <name>")
			}

			store := lockfile.NewFSStore(fs, "conduit.lock")
			args := conduitcli.DiffArgs{
				RootDir:              ".",
				MigrationsDir:        filepath.Clean(cmd.String(cmdutil.MigrationsDir)),
				Name:                 name,
				SchemaPath:           cmd.String(schemaFlag),
				DatabaseURL:          cmd.String(cmdutil.DatabaseURL),
				ExcludeSchemas:       cmd.StringSlice(cmdutil.ExcludeSchemas),
				SkipSchemaDriftCheck: cmd.Bool(cmdutil.SkipSchemaDriftCheck),
			}

			result, err := conduitcli.Diff(ctx, fs, timeGen, bi, store, args)
			if errors.Is(err, conduitcli.ErrNoChanges) {
				fmt.Fprintln(stderr, "No schema changes detected.")

				return nil
			}

			if err != nil {
				return fmt.Errorf("failed to generate diff: %w", err)
			}

			for _, f := range result.Files {
				fmt.Fprintln(stderr, "Created "+f.Path)
			}

			fmt.Fprintln(stderr, "Updated conduit.lock")

			return nil
		},
	}
}
