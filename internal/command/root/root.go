package root

import "github.com/urfave/cli/v2"

var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "dir",
		Usage:   "directory with migration files",
		Value:   "migrations",
		EnvVars: []string{"CONDUIT_MIGRATION_DIR"},
	},
	&cli.BoolFlag{
		Name:  "verbose",
		Usage: "verbose mode",
		Value: false,
	},
	&cli.StringFlag{
		Name:    "database-url",
		Usage:   "database connection URL",
		EnvVars: []string{"CONDUIT_DATABASE_URL"},
	},
}
