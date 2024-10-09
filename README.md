# conduit

An SQL migrator that is easy to embed.

## FAQ

<details>
<summary>Why a new migration tool when there are already Goose, golang-migrate, atlas, etc.?</summary>

While those tools are all excellent, they do not solve one specific problem - I wanted to build a migration tool specifically designed to create embeddable migrations. Even though Goose provides programming interface, unfortunately, it doesn't allow building isolated migrations similar to `rivermigrate`.

Check out my post where I cover the topic in more detail: https://romanvanesyan.com/articles/conduit

</details>

<details>
<summary>Will there be support for databases other than Postgres?</summary>

At this moment, it is unlikely that support will be added for any databases other than Postgres. conduit is built on top of the excellent Postgres driver - pgx, and uses specific Postgres functionality such as advisory locks.

</details>
