package migrationctx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"go.inout.gg/foundations/must"

	"go.inout.gg/conduit/internal/command/flagname"
	"go.inout.gg/conduit/internal/internaldebug"
)

type ctx struct{ _ int }

//nolint:gochecknoglobals
var kMigrationCtx = &ctx{1}

// OnBeforeHook sets up the migrations directory in the context.
// This should be called before any command that needs access to the migrations directory.
func OnBeforeHook(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	dir := cmd.String(flagname.MigrationsDir)
	if dir == "" {
		return ctx, fmt.Errorf("conduit: missing `%s' flag", flagname.MigrationsDir)
	}

	internaldebug.Log("supplied migrations directory: %s", dir)

	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		dir = filepath.Clean(filepath.Join(must.Must(os.Getwd()), dir))
	}

	internaldebug.Log("resolved migrations directory: %s", dir)

	ctx = context.WithValue(ctx, kMigrationCtx, dir)

	return ctx, nil
}

// Dir returns the migration directory from the context.
func Dir(ctx context.Context) (string, error) {
	if v, ok := ctx.Value(kMigrationCtx).(string); ok {
		internaldebug.Log("resolved migration directory: %s", v)
		return v, nil
	}

	return "", errors.New("conduit: failed to resolve migration directory")
}
