package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	dotenv "github.com/joho/godotenv"

	"go.inout.gg/conduit/cmd/internal/command"
)

func main() {
	_ = dotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	err := command.Execute(ctx)
	if err != nil {
		cancel()
		log.Fatal(err)
	}

	cancel()
}
