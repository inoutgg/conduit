package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	dotenv "github.com/joho/godotenv"

	"go.inout.gg/conduit/cmd/internal/command"
	"go.inout.gg/conduit/cmd/internal/conduiterror"
)

func main() {
	_ = dotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	err := command.Execute(ctx)
	if err != nil {
		cancel()
		conduiterror.Display(os.Stderr, err)
		os.Exit(1)
	}

	cancel()
}
