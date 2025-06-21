package common

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v3"
)

var (
	databaseURLFlagName   = "database-url"   //nolint:gochecknoglobals
	migrationsDirFlagName = "migrations-dir" //nolint:gochecknoglobals
)

//nolint:gochecknoglobals,exhaustruct
var DatabaseURLFlag = &cli.StringFlag{
	Name:     databaseURLFlagName,
	Usage:    "database connection URL",
	Sources:  cli.EnvVars("CONDUIT_DATABASE_URL"),
	Required: true,
}

//nolint:gochecknoglobals,exhaustruct
var MigrationsDirFlag = &cli.StringFlag{
	Name:    migrationsDirFlagName,
	Usage:   "directory with migration files",
	Value:   "migrations",
	Sources: cli.EnvVars("CONDUIT_MIGRATION_DIR"),
}

//nolint:gochecknoglobals,exhaustruct
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
func Conn(ctx context.Context, cmd *cli.Command) (*pgx.Conn, error) {
	url := cmd.String(databaseURLFlagName)
	if url == "" {
		return nil, fmt.Errorf("conduit: missing `%s' flag", databaseURLFlagName)
	}

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to connect to database: %w", err)
	}

	return conn, nil
}
