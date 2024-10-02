package root

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"
)

var DatabaseURLFlag = &cli.StringFlag{
	Name:     "database-url",
	Usage:    "database connection URL",
	EnvVars:  []string{"CONDUIT_DATABASE_URL"},
	Required: true,
}

var MigrationsDirFlag = &cli.StringFlag{
	Name:    "dir",
	Usage:   "directory with migration files",
	Value:   "migrations",
	EnvVars: []string{"CONDUIT_MIGRATION_DIR"},
}

var GlobalFlags = []cli.Flag{
	MigrationsDirFlag,
	&cli.BoolFlag{
		Name:  "verbose",
		Usage: "verbose mode",
		Value: false,
	},
}

func Conn(ctx *cli.Context) (*pgx.Conn, error) {
	url := ctx.String("database-url")
	if url == "" {
		return nil, fmt.Errorf("conduit: missing `database-url\" flag.")
	}

	return pgx.Connect(ctx.Context, url)
}
