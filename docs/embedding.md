# Embedding Migrations in a Go Application

Conduit is designed to run migrations directly from a Go binary. SQL files are
compiled into the binary using Go's `embed` package, so there is nothing extra
to deploy alongside the application.

## Basic setup

```go
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

	// Register the embedded SQL files in the global registry.
	conduit.FromFS(migrations, "migrations")

	// Create a migrator. With no options it uses the global registry
	// and slog.Default() for logging.
	migrator := conduit.NewMigrator()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	if _, err = migrator.Migrate(ctx, conduit.DirectionUp, conn, nil); err != nil {
		log.Fatal(err)
	}
}
```

`conduit.FromFS` accepts any `fs.FS`, so you can also pass a sub-filesystem or
an OS directory via `os.DirFS` during development.

## Using a private registry

`conduit.FromFS` populates a package-level global registry, which works well
for single-binary applications. If you need multiple independent registries —
for example in tests or in an application that manages several databases — use
`conduitregistry.FromFS` directly and pass the result to `WithRegistry`:

```go
import (
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/conduitregistry"
	"github.com/spf13/afero"
)

registry := conduitregistry.FromFS(afero.NewOsFs(), "./migrations")
migrator := conduit.NewMigrator(conduit.WithRegistry(registry))
```

## Options

`NewMigrator` accepts functional options:

| Option                        | Description                                        |
| ----------------------------- | -------------------------------------------------- |
| `WithRegistry(r)`             | Use a specific registry instead of the global one  |
| `WithLogger(l)`               | Use a custom `*slog.Logger` for debug output       |
| `WithExecutor(e)`             | Use a custom `MigrationExecutor`; defaults to `NewLiveExecutor` which applies migrations to the database. Use `NewDryRunExecutor` to preview migrations without applying them. |
| `WithSkipSchemaDriftCheck()`  | Skip the schema drift check before applying up migrations. |

## Migrate options

`Migrate` accepts a `*MigrateOptions` struct (pass `nil` for defaults):

| Field          | Default (up) | Default (down) | Description                                   |
| -------------- | ------------ | -------------- | --------------------------------------------- |
| `Steps`        | `-1` (all)   | `1`            | Number of migrations to apply; `-1` means all |
| `AllowHazards` | `nil`        | `nil`          | Hazard types to permit; use `HazardType*` constants. Migrations with unlisted hazard types are blocked. |

```go
result, err := migrator.Migrate(ctx, conduit.DirectionUp, conn, &conduit.MigrateOptions{
	Steps: 5,
	AllowHazards: []conduit.MigrationHazardType{
		conduit.HazardTypeIndexBuild,
		conduit.HazardTypeDeletesData,
	},
})
```

## Reading migration results

`Migrate` returns a `*MigrateResult` with the list of applied migrations and
their durations:

```go
result, err := migrator.Migrate(ctx, conduit.DirectionUp, conn, nil)
if err != nil {
	log.Fatal(err)
}

for _, m := range result.MigrationResults {
	fmt.Printf("applied %s_%s in %s\n", m.Version, m.Name, m.DurationTotal)
}
```

## Advisory locking

`Migrate` acquires a PostgreSQL advisory lock before running migrations, so it
is safe to call concurrently from multiple application instances — for example,
at startup in a horizontally scaled deployment. Only one instance will run the
migrations; the others will wait and then proceed once the lock is released.
