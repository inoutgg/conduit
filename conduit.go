// Package conduit provides embeddable PostgreSQL migrations for Go applications.
//
// Register SQL migration files from an [embed.FS] with [FromFS], create a
// [Migrator], and call [Migrator.Migrate] to apply or roll back migrations.
//
// Only pgx v5 is supported as a database driver.
//
//	conduit.FromFS(migrations, "migrations")
//	m := conduit.NewMigrator()
//	result, err := m.Migrate(ctx, conduit.DirectionUp, conn, nil)
package conduit
