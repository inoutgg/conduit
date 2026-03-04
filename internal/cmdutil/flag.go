package cmdutil

import (
	altsrc "github.com/urfave/cli-altsrc/v3"
	yamlsrc "github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
)

const (
	Verbose       = "verbose"
	DatabaseURL   = "database-url"
	MigrationsDir = "migrations-dir"
)

func MigrationsDirFlag(src altsrc.Sourcer) *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:  MigrationsDir,
		Usage: "directory with migration files",
		Value: "./migrations",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("CONDUIT_MIGRATIONS_DIR"),
			yamlsrc.YAML("migrations.dir", src),
		),
	}
}

func DatabaseURLFlag(src altsrc.Sourcer) *cli.StringFlag {
	//nolint:exhaustruct
	return &cli.StringFlag{
		Name:  DatabaseURL,
		Usage: "database connection URL",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("CONDUIT_DATABASE_URL"),
			yamlsrc.YAML("database.url", src),
		),
	}
}

func VerboseFlag(src altsrc.Sourcer) *cli.BoolFlag {
	//nolint:exhaustruct
	return &cli.BoolFlag{
		Name:  Verbose,
		Usage: "verbose mode",
		Sources: cli.NewValueSourceChain(
			cli.EnvVar("CONDUIT_VERBOSE"),
			yamlsrc.YAML("verbose", src),
		),
	}
}
