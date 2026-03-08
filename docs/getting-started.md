# Getting Started

This guide walks through initializing a conduit project, creating your first
migration, and applying it to a database.

## Prerequisites

- A running PostgreSQL instance
- The `conduit` CLI installed

Set the database URL once so every command picks it up:

```sh
export CONDUIT_DATABASE_URL="postgres://user:pass@localhost:5432/mydb"
```

Alternatively, pass `--database-url` to each command, or place the variable in a
`.env` file (conduit loads it automatically).

## 1. Initialize the project

```sh
conduit init
```

This creates a `migrations/` directory and three files:

- An initial migration that sets up conduit's internal `conduit_migrations`
  table.
- A `conduit.sum` file that tracks the expected schema hash for drift detection.
- A `conduit.yaml` config file.

To use a custom directory:

```sh
conduit init --migrations-dir ./db/migrations
```

## 2. Write a target schema

Conduit generates migrations by diffing a **target schema file** against the
current state of the migrations directory. Create a SQL file that describes the
desired database schema:

```sql
-- schema.sql
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 3. Generate a migration

### From a schema diff

```sh
conduit diff add_users --schema schema.sql
```

Conduit compares `migrations/` against `schema.sql` using a temporary database
and writes a new `.up.sql` file into `migrations/`. Statements that cannot run
inside a transaction (e.g. `CREATE INDEX CONCURRENTLY`) are automatically split
into separate migration files.

If the generated migration contains hazardous operations (e.g. acquiring an
access-exclusive lock or deleting data), they are annotated with
`---- hazard: <TYPE> // <reason> ----` comments so you can review them before
applying. Conduit will refuse to run such a migration unless you explicitly
allow the relevant hazard types — see [Hazardous operations](#hazardous-operations).

### As an empty file

When you need to write a migration by hand — for example to seed data or run a
stored procedure — use `conduit new` instead:

```sh
conduit new seed_users
```

This creates a pair of empty `<timestamp>_seed_users.up.sql` and
`<timestamp>_seed_users.down.sql` files in the migrations directory, ready for
you to fill in.

## 4. Apply migrations

Roll forward all pending migrations:

```sh
conduit apply up
```

Roll back one migration:

```sh
conduit apply down
```

> **Note:** `conduit diff` only generates `.up.sql` files. Down migrations are
> not created automatically and must be written by hand if needed. In practice,
> rolling back is rarely the right response to a problem in production — a new
> forward migration that corrects the issue is safer and keeps history intact.
> Reserve `apply down` for local development.

### Options

| Flag                          | Description                                              |
| ----------------------------- | -------------------------------------------------------- |
| `--steps N`                   | Limit the number of migrations to run                    |
| `--allow-hazards HAZARD_TYPE` | Allow a specific hazard type; may be repeated            |
| `--skip-schema-drift-check`   | Skip schema drift detection                              |
| `--dry-run`                   | Preview migrations without applying them                 |

## Hazardous operations

Some schema changes carry operational risk — for example, adding a column with a
non-constant default rewrites every row and holds an access-exclusive lock for
the duration. Conduit blocks such migrations by default and requires you to
explicitly opt in to each hazard type you want to allow.

Hazard types are defined and documented by
[pg-schema-diff](https://github.com/stripe/pg-schema-diff). Pass the string
value of the type to `--allow-hazards` (the flag may be repeated):

```sh
conduit apply up --allow-hazards ACQUIRES_ACCESS_EXCLUSIVE_LOCK
conduit apply up --allow-hazards INDEX_BUILD --allow-hazards DELETES_DATA
```

See [embedding.md](embedding.md#hazard-types) for the full list of constants
available when embedding conduit in a Go application.

## 5. Embed migrations in your Go application

See [embedding.md](embedding.md) for how to run migrations from within your Go
application.

## Iterating on the schema

The typical workflow after the initial setup:

1. Edit `schema.sql` with the desired changes.
2. Run `conduit diff <name> --schema schema.sql` to generate a migration.
3. Review the generated `.up.sql` file.
4. Run `conduit apply up`.
