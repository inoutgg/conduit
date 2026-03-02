package conduit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stripe/pg-schema-diff/pkg/schema"
	"go.inout.gg/foundations/must"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
)

// MigrationExecutor executes a single migration.
type MigrationExecutor interface {
	Execute(
		context.Context,
		*conduitregistry.Migration,
		Direction,
		*pgx.Conn,
	) (MigrationResult, error)
}

// NewLiveExecutor returns an executor that applies migrations to the database.
func NewLiveExecutor(logger *slog.Logger) MigrationExecutor {
	return &liveExecutor{logger: logger}
}

// NewDryRunExecutor returns an executor that logs migrations to w without
// applying them. When verbose is true, the migration SQL content is also
// written.
func NewDryRunExecutor(w io.Writer, verbose bool) MigrationExecutor {
	return &dryRunExecutor{w: w, verbose: verbose}
}

// liveExecutor applies migrations to the database.
type liveExecutor struct {
	logger *slog.Logger
}

func (e *liveExecutor) Execute(
	ctx context.Context,
	migration *conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
) (MigrationResult, error) {
	inTx := must.Must(migration.UseTx(dir))

	e.logger.DebugContext(
		ctx,
		"applying migration",
		slog.String("direction", string(dir)),
		slog.Group(
			"migration",
			slog.String("version", migration.Version().String()),
			slog.String("name", migration.Name()),
		),
		slog.Bool("transacting", inTx),
	)

	start := time.Now()

	var err error
	if inTx {
		err = applyMigrationTx(ctx, migration, dir, conn)
	} else {
		err = migration.Apply(ctx, dir, conn)
	}

	if err != nil {
		return MigrationResult{}, fmt.Errorf(
			"conduit: failed to apply migration %s: %w",
			migration.Version().String(),
			err,
		)
	}

	duration := time.Since(start)
	result := MigrationResult{
		DurationTotal: duration,
		Version:       migration.Version(),
		Name:          migration.Name(),
	}

	switch dir {
	case DirectionDown:
		err = dbsqlc.New().RollbackMigration(ctx, conn, dbsqlc.RollbackMigrationParams{
			Version: result.Version.String(),
			Name:    result.Name,
		})

	case DirectionUp:
		var schemaHash string

		schemaHash, err = computeSchemaHash(ctx, conn)
		if err != nil {
			return MigrationResult{}, fmt.Errorf(
				"conduit: failed to compute schema hash after migration %s: %w",
				migration.Version().String(),
				err,
			)
		}

		err = dbsqlc.New().ApplyMigration(ctx, conn, dbsqlc.ApplyMigrationParams{
			Version: result.Version.String(),
			Name:    result.Name,
			Hash:    schemaHash,
		})
	}

	if err != nil {
		return MigrationResult{}, fmt.Errorf("conduit: failed to update migrations table %v: %w", dir, err)
	}

	_ = dbsqlc.New().ResetConn(ctx, conn)

	return result, nil
}

// dryRunExecutor logs migrations without applying them.
type dryRunExecutor struct {
	w       io.Writer
	verbose bool
}

func (e *dryRunExecutor) Execute(
	_ context.Context,
	migration *conduitregistry.Migration,
	dir Direction,
	_ *pgx.Conn,
) (MigrationResult, error) {
	fmt.Fprintf(e.w, "%s_%s\n", migration.Version().String(), migration.Name())

	if e.verbose {
		fmt.Fprintf(e.w, "%s\n", migration.Content(dir))
	}

	//nolint:exhaustruct
	return MigrationResult{
		Version: migration.Version(),
		Name:    migration.Name(),
	}, nil
}

func applyMigrationTx(
	ctx context.Context,
	migration *conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("conduit: failed to open transaction: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if err := migration.ApplyTx(ctx, dir, tx); err != nil {
		//nolint:wrapcheck
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("conduit: failed to commit transaction: %w", err)
	}

	return nil
}

func computeSchemaHash(ctx context.Context, conn *pgx.Conn) (string, error) {
	db := stdlib.OpenDB(*conn.Config())
	defer db.Close()

	hash, err := schema.GetSchemaHash(ctx, db)
	if err != nil {
		return "", fmt.Errorf("conduit: failed to compute schema hash: %w", err)
	}

	return hash, nil
}
