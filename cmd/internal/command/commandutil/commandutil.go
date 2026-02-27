package commandutil

import "github.com/urfave/cli/v3"

const (
	Verbose       = "verbose"
	DatabaseURL   = "database-url"
	MigrationsDir = "migrations-dir"
	PackageName   = "package-name"
)

func MigrationsDirFlag() *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:    MigrationsDir,
		Usage:   "directory with migration files",
		Value:   "./migrations",
		Sources: cli.EnvVars("CONDUIT_MIGRATION_DIR"),
	}
}

func DatabaseURLFlag(required bool) *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:     DatabaseURL,
		Usage:    "database connection URL",
		Sources:  cli.EnvVars("CONDUIT_DATABASE_URL"),
		Required: required,
	}
}
