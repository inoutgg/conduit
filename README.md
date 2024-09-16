# conduit

An SQL migrator that is easy to embed.

## FAQ

<details>
<summary>Why a new migration tooling when there are already Goose, golang-migrate, atlas, etc.?</summary>

While those tools are all exciting, they do not solve one problem - I'd like to build a migration tool specifically to be able to create embeddable migrations. Even-though Goose gives some direct programming interface, unfortunately, it doesn't allow to build isolated migrations similar to `rivermigrate`.

Checkout my post where I cover the topic in more details: https://romanvanesyan.com/articles/conduit

</details>

<details>
<summary>Will there be any support for databases other than Postgres?</summary>

At this moment, it is quite unlikely that support will be added for any databases other than Postgres. conduit is built on top of excellent Postgres driver - pgx and uses some specific Postgres functionality such as advisory locks.

</details>
