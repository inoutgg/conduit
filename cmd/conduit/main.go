package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	dotenv "github.com/joho/godotenv"
	"github.com/spf13/afero"

	"go.inout.gg/conduit/cmd/internal/command"
	"go.inout.gg/conduit/cmd/internal/conduiterror"
	"go.inout.gg/conduit/pkg/conduitbuildinfo"
	"go.inout.gg/conduit/pkg/stopwatch"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func main() {
	_ = dotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	if err := run(ctx, os.Stdout, os.Stderr, os.Args); err != nil {
		cancel()
		conduiterror.Display(os.Stderr, err)
		os.Exit(1)
	}

	cancel()
}

func run(ctx context.Context, stdout io.Writer, stderr io.Writer, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var (
		timeGen timegenerator.Standard
		bi      conduitbuildinfo.Standard
		sw      stopwatch.Standard
		fs      = afero.NewBasePathFs(afero.NewOsFs(), cwd)
	)

	//nolint:wrapcheck
	return command.Execute(ctx, fs, stdout, stderr, timeGen, bi, sw, cwd, args)
}
