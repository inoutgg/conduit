package commandutil

import "github.com/urfave/cli/v3"

const (
	Verbose       = "verbose"
	DatabaseURL   = "database-url"
	MigrationsDir = "migrations-dir"
)

func MigrationsDirFlag() *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:    MigrationsDir,
		Usage:   "directory with migration files",
		Value:   "./migrations",
		Sources: cli.EnvVars("CONDUIT_MIGRATIONS_DIR"),
	}
}

func DatabaseURLFlag() *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:    DatabaseURL,
		Usage:   "database connection URL",
		Sources: cli.EnvVars("CONDUIT_DATABASE_URL"),
	}
}

func VerboseFlag(usage string) *cli.BoolFlag {
	//nolint:exhaustruct
	return &cli.BoolFlag{
		Name:    Verbose,
		Usage:   usage,
		Sources: cli.EnvVars("CONDUIT_VERBOSE"),
	}
}
