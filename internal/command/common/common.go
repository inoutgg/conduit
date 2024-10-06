package common

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"
)

var (
	databaseUrlFlagName = "database-url"
	migrationsDirFlagName         = "dir"
)

var DatabaseURLFlag = &cli.StringFlag{
	Name:     databaseUrlFlagName,
	Usage:    "database connection URL",
	EnvVars:  []string{"CONDUIT_DATABASE_URL"},
	Required: true,
}

var MigrationsDirFlag = &cli.StringFlag{
	Name:    migrationsDirFlagName,
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

// Conn attempts to connect to the database available at provided
// `--database-url` flag.
func Conn(ctx *cli.Context) (*pgx.Conn, error) {
	url := ctx.String(databaseUrlFlagName)
	if url == "" {
		return nil, fmt.Errorf("conduit: missing `%s\" flag.", databaseUrlFlagName)
	}

	return pgx.Connect(ctx.Context, url)
}
