package initialise

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/config"
	"go.inout.gg/conduit/conduitcli"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func NewCommand(fs afero.Fs, timeGen timegenerator.Generator, cfg *config.Config) *cli.Command {
	//nolint:exhaustruct
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "initialise migration directory",
		Flags: []cli.Flag{
			cmdutil.MigrationsDirFlag(),
			cmdutil.DatabaseURLFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dirPath, _ := config.FilePath(cfg.Migrations.Dir)
			migrationsDir := cmdutil.StringOr(cmd, cmdutil.MigrationsDir, dirPath)
			dbURL := cmdutil.StringOr(cmd, cmdutil.DatabaseURL, cfg.Database.URL)

			args := conduitcli.InitArgs{
				Dir:         filepath.Clean(migrationsDir),
				DatabaseURL: dbURL,
			}

			if err := conduitcli.Init(ctx, fs, timeGen, args); err != nil {
				return fmt.Errorf("conduit: init: %w", err)
			}

			defaultCfg := config.Config{
				Migrations: config.MigrationsConfig{
					Dir: &url.URL{Scheme: "file", Path: args.Dir},
				},
				Database: config.DatabaseConfig{
					URL: args.DatabaseURL,
				},
			}

			if err := config.WriteFile(fs, config.DefaultFilename, defaultCfg); err != nil {
				return fmt.Errorf("conduit: failed to create config file: %w", err)
			}

			return nil
		},
	}
}
