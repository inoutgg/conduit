package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"go.inout.gg/foundations/must"

	"go.inout.gg/conduit/internal/internaldebug"
)

type ctx struct{}

//nolint:gochecknoglobals
var kCtx = &ctx{}

func OnBeforeHook(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Attach migration directory to the context.
	migrationsDir := cmd.String(migrationsDirFlagName)
	if migrationsDir == "" {
		return ctx, fmt.Errorf("conduit: missing `%s' flag", migrationsDirFlagName)
	}

	internaldebug.Log("supplied migrations directory: %s", migrationsDir)

	migrationsDir = filepath.Clean(migrationsDir)
	if !filepath.IsAbs(migrationsDir) {
		migrationsDir = filepath.Clean(filepath.Join(must.Must(os.Getwd()), migrationsDir))
	}

	internaldebug.Log("resolved migrations directory: %s", migrationsDir)

	ctx = context.WithValue(ctx, kCtx, migrationsDir)

	return ctx, nil
}

// MigrationDir returns the migration directory.
//
// It resolves the migration directory from the current working directory.
func MigrationDir(ctx context.Context) (string, error) {
	if v, ok := ctx.Value(kCtx).(string); ok {
		internaldebug.Log("resolved migration directory: %s", v)
		return v, nil
	}

	return "", errors.New("conduit: failed to resolve migration directory")
}
