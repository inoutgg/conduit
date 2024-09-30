package conduit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/direction"
	internalmigrations "go.inout.gg/conduit/internal/migrations"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/internal/uuidv7"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"
)

var _ Migrator = (*migrator)(nil)

// A concatenation of encoded "migrations" letters by the following mapping: a=1,b=2,...z=26
const pgLockNum = int64(13971812091514)

const (
	DirectionUp   Direction = direction.DirectionUp
	DirectionDown           = direction.DirectionDown
)

type (
	// Direction denotes whether SQL migration should be rolled up, or rolled back.
	Direction = direction.Direction

	MigrateFunc   = conduitregistry.MigrateFunc
	MigrateFuncTx = conduitregistry.MigrateFuncTx
	Migration     = conduitregistry.Migration
)

type Migrator interface {
	Migrate(context.Context, Direction, *pgx.Conn) (*MigrateResult, error)
}

type Config struct {
	Logger *slog.Logger

	// Migration conduitregistry.
	Registry *conduitregistry.Registry
}

// NewConfig creates a new Config which can be optionally updated via cfgs.
// If Config.Registry is not provided it is falls back to the Global Registry.
func NewConfig(cfgs ...func(*Config)) *Config {
	config := &Config{}
	for _, c := range cfgs {
		c(config)
	}

	config.defaults()

	return config
}

// defaults applies default configurations.
func (c *Config) defaults() {
	if c.Logger == nil {
		c.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	c.Registry = cmp.Or(c.Registry, globalRegistry)
}

// MigrateResult is the result for a applied batch migrations.
type MigrateResult struct {
	Direction        direction.Direction
	MigrationResults []MigrationResult
}

// MigrationResult is the result for a single applied migration.
type MigrationResult struct {
	// Total time it took to apply migration
	DurationTotal time.Duration

	// Name of the applied migration
	Name string

	// Namespace of the applied migration
	Namespace string

	// Version of the applied migration
	Version int64
}

func NewMigrator(config *Config) Migrator {
	debug.Assert(config.Logger != nil, "config.Logger must be defined")
	debug.Assert(config.Registry != nil, "config.Registry must be defined")

	return &migrator{
		logger:   config.Logger,
		registry: config.Registry,
	}
}

type migrator struct {
	logger   *slog.Logger
	registry *conduitregistry.Registry
}

// existingMigrationVerions returns a list of migration versions that
// have been already applied.
func (m *migrator) existingMigrationVerions(ctx context.Context, conn *pgx.Conn) ([]int64, error) {
	ok, err := dbsqlc.New().DoesTableExist(ctx, conn, "conduitmigrations")
	if err != nil {
		return nil, fmt.Errorf("conduit: unable to fetch info about migrations table: %w", err)
	}

	if !ok {
		return []int64{}, nil
	}

	versions, err := dbsqlc.New().AllExistingMigrationVersions(ctx, conn, m.registry.Namespace)
	if err != nil {
		return nil, fmt.Errorf("conduit: unable to fetch existing versions", err)
	}

	return versions, nil
}

func (m *migrator) Migrate(
	ctx context.Context,
	direction Direction,
	conn *pgx.Conn,
) (result *MigrateResult, err error) {
	debug.Assert(conn != nil, "expected conn to be defined")

	if err := dbsqlc.New().AcquireLock(ctx, conn, pgLockNum); err != nil {
		return nil, fmt.Errorf("conduit: unable to acquire a lock: %w", err)
	}
	defer dbsqlc.New().ReleaseLock(ctx, conn, pgLockNum)

	switch direction {
	case DirectionUp:
		result, err = m.migrateUp(ctx, conn)
	case DirectionDown:
		result, err = m.migrateDown(ctx, conn)
	default:
		return nil, errors.New("conduit: unknown direction, expected either up or down")
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *migrator) migrateUp(ctx context.Context, conn *pgx.Conn) (*MigrateResult, error) {
	existingMigrationVersions, err := m.existingMigrationVerions(ctx, conn)
	if err != nil {
		return nil, err
	}

	targetMigrations := m.registry.Migrations()
	for _, existingVersion := range existingMigrationVersions {
		delete(targetMigrations, existingVersion)
	}
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return int(a.Version() - b.Version())
	})

	return m.applyMigrations(ctx, migrations, DirectionUp, conn)
}

func (m *migrator) migrateDown(ctx context.Context, conn *pgx.Conn) (*MigrateResult, error) {
	existingMigrations, err := m.existingMigrationVerions(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Populate only already applied migrations.
	existingMigrationsMap := sliceutil.KeyBy(existingMigrations, func(e int64) int64 { return e })
	targetMigrations := m.registry.Migrations()
	for _, m := range targetMigrations {
		if _, ok := existingMigrationsMap[m.Version()]; !ok {
			delete(targetMigrations, m.Version())
		}
	}

	// Sort in descending order, as we need to roll back starting from the
	// last applied migration tMigrationTy first one.
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return int(b.Version() - a.Version())
	})

	return m.applyMigrations(ctx, migrations, DirectionDown, conn)
}

func (m *migrator) applyMigrations(
	ctx context.Context,
	migrations []*conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
) (result *MigrateResult, err error) {
	results := make([]MigrationResult, len(migrations))
	for i, migration := range migrations {
		var tx pgx.Tx
		inTx := must.Must(migration.UseTx(dir))

		m.logger.Info(
			"applying migration",
			slog.String("direction", string(dir)),
			slog.Group(
				"migration",
				slog.Int64("version", migration.Version()),
				slog.String("name", migration.Name()),
			),
			slog.Bool("transacting", inTx),
		)

		if inTx {
			tx, err = conn.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf(
					"conduit: an error occurred while opening transaction: %w",
					err,
				)
			}
			defer tx.Rollback(ctx)
		}

		start := time.Now()
		if err := migration.Apply(ctx, dir, conn, tx); err != nil {
			return nil, fmt.Errorf(
				"conduit: an error occurred while applying migration %d: %w",
				migration.Version(),
				err,
			)
		}

		duration := time.Since(start)
		migrationResult := MigrationResult{
			DurationTotal: duration,
			Version:       migration.Version(),
			Name:          migration.Name(),
			Namespace:     m.registry.Namespace,
		}
		results[i] = migrationResult

		switch dir {
		case DirectionDown:
			// "migrations" table won't be available by this time.
			isFirstConduitMigration := migrationResult.Version == int64(internalmigrations.FirstMigrationVersion) && migrationResult.Namespace == internalmigrations.RegistryNamespace
			if isFirstConduitMigration {
				break
			}

			err = dbsqlc.New().RollbackMigration(ctx, conn, dbsqlc.RollbackMigrationParams{
				Version:   migrationResult.Version,
				Namespace: migrationResult.Namespace,
			})

		case DirectionUp:
			err = dbsqlc.New().ApplyMigration(ctx, conn, dbsqlc.ApplyMigrationParams{
				ID:        uuidv7.Must(),
				Version:   migrationResult.Version,
				Namespace: migrationResult.Namespace,
				Name:      migrationResult.Name,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("conduit: an error occurred while updating migrations table (%v): %w", dir, err)
		}

		_ = dbsqlc.New().ResetConn(ctx, conn)

		if inTx {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf(
					"conduit: an error occurred while committing migration %d tx: %w",
					migration.Version(),
					err,
				)
			}
		}
	}

	result = &MigrateResult{
		MigrationResults: results,
		Direction:        DirectionDown,
	}

	return result, err
}
