package command

import (
	"context"
	"io"
	"path/filepath"

	"github.com/spf13/afero"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"

	"go.inout.gg/conduit/cmd/internal/command/apply"
	"go.inout.gg/conduit/cmd/internal/command/diff"
	"go.inout.gg/conduit/cmd/internal/command/dump"
	"go.inout.gg/conduit/cmd/internal/command/initialise"
	newmigration "go.inout.gg/conduit/cmd/internal/command/new"
	"go.inout.gg/conduit/internal/cmdutil"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
	"go.inout.gg/conduit/pkg/stopwatch"
	"go.inout.gg/conduit/pkg/timegenerator"
)

// Execute evaluates given os.Args and executes a matched command.
func Execute(
	ctx context.Context,
	fs afero.Fs,
	stdout io.Writer,
	stderr io.Writer,
	timeGen timegenerator.Generator,
	bi conduitbuildinfo.BuildInfo,
	timer stopwatch.Stopwatch,
	rootDir string,
	args []string,
) error {
	configPath := filepath.Join(rootDir, "conduit.yaml")
	configSrc := altsrc.NewStringPtrSourcer(&configPath)

	//nolint:exhaustruct
	cmd := &cli.Command{
		Name:  "conduit",
		Usage: "An SQL migrator that is easy to embed.",
		Flags: []cli.Flag{
			//nolint:exhaustruct
			&cli.StringFlag{
				Name:        "config",
				Usage:       "path to config file",
				Value:       "conduit.yaml",
				Destination: &configPath,
				Sources:     cli.EnvVars("CONDUIT_CONFIG"),
			},
			cmdutil.VerboseFlag(configSrc),
		},
		Commands: []*cli.Command{
			initialise.NewCommand(fs, stdout, stderr, timeGen, configSrc),
			newmigration.NewCommand(fs, stdout, stderr, timeGen, configSrc),
			diff.NewCommand(fs, stdout, stderr, timeGen, bi, configSrc),
			apply.NewCommand(fs, stdout, stderr, timer, configSrc),
			dump.NewCommand(stdout, bi, configSrc),
		},
	}

	//nolint:wrapcheck
	return cmd.Run(ctx, args)
}
