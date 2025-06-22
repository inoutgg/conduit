package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"go.inout.gg/foundations/must"
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

	migrationsDir = filepath.Clean(migrationsDir)
	if !filepath.IsAbs(migrationsDir) {
		migrationsDir = filepath.Clean(filepath.Join(must.Must(os.Getwd()), migrationsDir))
	}

	ctx = context.WithValue(ctx, kCtx, migrationsDir)

	return ctx, nil
}

func MigrationDir(ctx context.Context) (string, error) {
	if v, ok := ctx.Value(kCtx).(string); ok {
		return v, nil
	}

	return "", errors.New("conduit: failed to resolve migration directory")
}
