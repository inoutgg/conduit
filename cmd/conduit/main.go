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
	"go.inout.gg/conduit/pkg/buildinfo"
	"go.inout.gg/conduit/pkg/timegenerator"
)

func main() {
	_ = dotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	if err := run(ctx, os.Stdout, os.Args); err != nil {
		cancel()
		conduiterror.Display(os.Stderr, err)
		os.Exit(1)
	}

	cancel()
}

func run(ctx context.Context, w io.Writer, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	var (
		timeGen timegenerator.Standard
		bi      buildinfo.Standard
		fs      = afero.NewBasePathFs(afero.NewOsFs(), cwd)
	)

	if err := command.Execute(ctx, fs, w, timeGen, bi, cwd, args); err != nil {
		//nolint:wrapcheck
		return err
	}

	return nil
}
