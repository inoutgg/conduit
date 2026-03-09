# conduit

An embeddable PostgreSQL migration tool for Go. Built on [pgx](https://github.com/jackc/pgx) and [pg-schema-diff](https://github.com/stripe/pg-schema-diff).

- **Embeddable** — ship migrations inside your Go binary with `go:embed`
- **Minimal-downtime schema diffing** — auto-generate migrations from a target schema via [pg-schema-diff](https://github.com/stripe/pg-schema-diff) with concurrent index builds, lock/statement timeouts, and online constraint handling
- **Drift detection** — catch out-of-band DDL changes before they cause problems
- **Hazard awareness** — surface risky operations (locks, data loss) before they run

```go
//go:embed migrations/*.sql
var migrations embed.FS

func main() {
    conduit.FromFS(migrations, "migrations")
    m := conduit.NewMigrator()

    result, err := m.Migrate(ctx, conduit.DirectionUp, conn, nil)
}
```

## Documentation

See [`go.inout.gg/conduit`](https://pkg.go.dev/go.inout.gg/conduit) for the full Go API reference.

### CLI quick reference

```
conduit init                          # scaffold a new project
conduit new <name>                    # create empty migration pair
conduit diff <name> --schema file.sql # generate migration from schema diff
conduit apply up                      # apply pending migrations
conduit apply down                    # roll back last migration
conduit apply up --dry-run            # preview without applying
conduit dump                          # dump current database schema
```

Run `conduit --help` for flags, env vars, and config file options.

### Migration files

```
migrations/
  20240101120000_create_users.up.sql
  20240101120000_create_users.down.sql
```

SQL comment directives control per-migration behavior:

```sql
---- enable-tx ----
---- hazard: INDEX_BUILD // rebuilds index ----
```

## FAQ

<details>
<summary>Why conduit over Goose, golang-migrate, or atlas?</summary>

Conduit is purpose-built for PostgreSQL and designed to embed into Go libraries and frameworks — not just applications.

It uses Postgres-specific capabilities throughout: [advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) for safe concurrent deploys, non-transactional migrations by default for online DDL like `CREATE INDEX CONCURRENTLY`, and [pg-schema-diff](https://github.com/stripe/pg-schema-diff) for generating minimal-downtime migrations with concurrent index builds, online constraint handling, lock/statement timeouts, and automatic hazard detection.

Embedding is a first-class concern: a library like an auth framework can ship its own migrations via `go:embed`, expose a `conduitregistry.Registry`, and let the host application compose registries — without requiring a CLI, external files, or runtime filesystem access.

The need for conduit emerged during development of [shield](https://github.com/inoutgg/shield), an authentication framework with deep database integration. For more background see https://segfaultmedaddy.com/p/conduit

</details>

<details>
<summary>Will there be support for databases other than Postgres?</summary>

No. Conduit is built around pgx and relies on Postgres-specific features like advisory locks, schema introspection, and pg-schema-diff.

</details>
