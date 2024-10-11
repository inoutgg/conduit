# conduit

An SQL migrator that is easy to embed.

## Getting started

### Creating a new Conduit migration project

To create a new conduit project in the current directory, run:

```
$ conduit init
```

By default, conduit initializes a new project in `$(cwd)/migrations`. To create the project in a different location, use:

```
$ conduit init --migrations-dir path/to/project
```

## FAQ

<details>
<summary>Why create conduit when tools like Goose, golang-migrate, and atlas already exist?</summary>

While existing tools are great for general-purpose migrations, conduit is designed specifically for building custom, embeddable migration frameworks.

The need for conduit emerged when during development of [shield](https://github.com/inoutgg/shield), an authentication framework with deep database integration.

For more detailed exploration of conduit's motivations check out https://romanvanesyan.com/articles/conduit

</details>

<details>
<summary>Will there be support for databases other than Postgres?</summary>

At this moment, it is unlikely that support for databases other than Postgres will be added. conduit is built around pgx, a robust Go Postgres driver, and utilizes Postgres-specific features like advisory locks to manage migrations.

</details>
