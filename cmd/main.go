package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.inout.gg/conduit/conduitcli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	config, err := conduitcli.ConfigFromEnv()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}

	cli := conduitcli.New(config)
	if err := cli.Execute(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
}
