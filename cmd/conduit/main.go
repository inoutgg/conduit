package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	dotenv "github.com/joho/godotenv"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitcli"
)

func main() {
	_ = dotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	dialer := conduit.NewDialer(func(conn *pgx.Conn) (conduit.Migrator, error) {
		return conduit.NewMigrator(conn, conduit.NewConfig()), nil
	})

	cli := conduitcli.New(dialer)
	if err := cli.Execute(ctx); err != nil {
		log.Fatal(err)
		os.Exit(1)
		return
	}
}
