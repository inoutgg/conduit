package root

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"go.inout.gg/foundations/must"
)

type ctx struct{}

var kCtx = &ctx{}

func OnBeforeHook(ctx *cli.Context) error {
	// Attach migration directory to the context.
	migrationsDir := ctx.String("dir")
	if migrationsDir == "" {
		return fmt.Errorf("conduit: expected migration directory to be provided.")
	}

	resolvedMigrationDir := filepath.Clean(filepath.Join(must.Must(os.Getwd()), ctx.String("dir")))
	ctx.Context = context.WithValue(ctx.Context, kCtx, resolvedMigrationDir)

	return nil
}

func MigrationDir(ctx *cli.Context) (string, error) {
	if v, ok := ctx.Context.Value(kCtx).(string); ok {
		return v, nil
	}

	return "", errors.New("conduit: failed to resolve migration directory")
}
