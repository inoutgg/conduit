package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"go.inout.gg/conduit"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	ctx := context.Background()

	conduit.FromFS(migrations, "migrations")

	migrator := conduit.NewMigrator()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	seq, err := migrator.Migrate(ctx, conduit.DirectionUp, conn, nil)
	if err != nil {
		conn.Close(ctx)
		log.Fatal(err)
	}

	for m, err := range seq {
		if err != nil {
			conn.Close(ctx)
			log.Fatal(err)
		}

		log.Printf("applied %s_%s (%s)", m.Version, m.Name, m.DurationTotal) //#nosec G706
	}

	conn.Close(ctx)
}
