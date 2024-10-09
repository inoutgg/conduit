package conduit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/internal/uuidv7"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"
)

const (
	allSteps = -1

	defaultUpStep   = allSteps
	defaultDownStep = 1
)

const (
	DirectionUp   Direction = direction.DirectionUp
	DirectionDown           = direction.DirectionDown
)

var ErrInvalidStep = errors.New(
	"conduit: invalid migration step. Expected: -1 (all) or positive integer.",
)

type (
	// Direction denotes whether SQL migration should be rolled up, or rolled back.
	Direction = direction.Direction

	MigrateFunc   = conduitregistry.MigrateFunc
	MigrateFuncTx = conduitregistry.MigrateFuncTx
	Migration     = conduitregistry.Migration
)

// Config is the configuration for the Migrator.
//
// Use NewConfig to instantiate a new instance.
//
// Logger and Registry fields are optional.
// If Registry is omitted, the global registry is used.
// If Logger is omitted, slog.Default is used.
type Config struct {
	Logger   *slog.Logger              // optional
	Registry *conduitregistry.Registry // optional
}

// WithLogger adds a logger to the Config.
//
// It's provided for convenience and intended to be used with NewConfig.
func WithLogger(l *slog.Logger) func(*Config) {
	return func(c *Config) { c.Logger = l }
}

// NewConfig creates a new Config and applies the provided configurations.
//
// If Config.Registry is not provided, it falls back to the global registry.
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
		c.Logger = slog.Default()
	}

	c.Registry = cmp.Or(c.Registry, globalRegistry)
}

// MigrateResult represents the outcome of applied migrations batch.
type MigrateResult struct {
	// Direction of the applied migrations.
	Direction direction.Direction

	// MigrationResults is a
	MigrationResults []MigrationResult
}

// MigrationResult represents the outcome of a single applied migration.
type MigrationResult struct {
	// Total time it took to apply migration
	DurationTotal time.Duration
	Name          string
	Namespace     string
	Version       int64
}

// MigrateOptions specifies options for a Migrator.Migrate operation.
type MigrateOptions struct {
	Steps int
}

func (m *MigrateOptions) validate() error {
	if !(m.Steps == -1 || m.Steps > 0) {
		return ErrInvalidStep
	}

	return nil
}

// NewMigrator creates a new migrator with the given config.
func NewMigrator(config *Config) *Migrator {
	debug.Assert(config.Logger != nil, "config.Logger must be defined")
	debug.Assert(config.Registry != nil, "config.Registry must be defined")

	return &Migrator{
		logger:   config.Logger,
		registry: config.Registry,
	}
}

// Migrator is a database migration tool that can rolls up and down migrations
// in order.
type Migrator struct {
	logger   *slog.Logger
	registry *conduitregistry.Registry
}

// existingMigrationVerions retrieves a list of already applied migration versions.
func (m *Migrator) existingMigrationVerions(ctx context.Context, conn *pgx.Conn) ([]int64, error) {
	ok, err := dbsqlc.New().DoesTableExist(ctx, conn, "conduitmigrations")
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch from migrations table: %w", err)
	}

	if !ok {
		d("conduitmigrations table is not found")
		return []int64{}, nil
	}

	versions, err := dbsqlc.New().AllExistingMigrationVersions(ctx, conn, m.registry.Namespace)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch existing versions: %w", err)
	}

	return versions, nil
}

// Migrate applies migrations in the specified direction (up or down).
//
// It uses a Postgres advisory lock before running migrations.
//
// By default, it applies all pending migrations when rolling up, and
// only one migration when rolling back. Use MigrateOptions.Step to control
// the number of migrations, or set it to -1 to migrate all.
//
// If a migration is registered in transaction mode, it creates a new transaction
// before applying the migration.
func (m *Migrator) Migrate(
	ctx context.Context,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (result *MigrateResult, err error) {
	debug.Assert(conn != nil, "expected conn to be defined")

	if opts == nil {
		opts = &MigrateOptions{Steps: defaultUpStep}
		if dir == DirectionDown {
			opts.Steps = defaultDownStep
		}

		d("opts is ommitted, using the default one: %v", opts)
	}

	if err := opts.validate(); err != nil {
		return nil, err
	}

	lockNum := pgLockNum(m.registry.Namespace)

	if err := dbsqlc.New().AcquireLock(ctx, conn, lockNum); err != nil {
		return nil, fmt.Errorf("conduit: failed to acquire a lock: %w", err)
	}
	defer dbsqlc.New().ReleaseLock(ctx, conn, lockNum)

	switch dir {
	case DirectionUp:
		result, err = m.migrateUp(ctx, conn, opts)
	case DirectionDown:
		result, err = m.migrateDown(ctx, conn, opts)
	default:
		return nil, direction.ErrUnknownDirection
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

// migrateUp applies pending migrations in the up direction.
//
// Migrations are rolled up in ascending order.
func (m *Migrator) migrateUp(
	ctx context.Context,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (*MigrateResult, error) {
	existingMigrationVersions, err := m.existingMigrationVerions(ctx, conn)
	if err != nil {
		return nil, err
	}

	targetMigrations := m.registry.CloneMigrations()
	for _, existingVersion := range existingMigrationVersions {
		delete(targetMigrations, existingVersion)
	}
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return int(a.Version() - b.Version())
	})

	return m.applyMigrations(ctx, migrations, DirectionUp, conn, opts)
}

// migrateDown rolls back applied migrations in the down direction.
//
// Migrations are rolled back in descending order.
func (m *Migrator) migrateDown(
	ctx context.Context,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (*MigrateResult, error) {
	existingMigrations, err := m.existingMigrationVerions(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Filter only applied migrations.
	existingMigrationsMap := sliceutil.KeyBy(existingMigrations, func(e int64) int64 { return e })
	targetMigrations := m.registry.CloneMigrations()
	for _, m := range targetMigrations {
		if _, ok := existingMigrationsMap[m.Version()]; !ok {
			delete(targetMigrations, m.Version())
		}
	}

	// Sort in descending order, as we need to roll back starting from the
	// last applied migration to the first one.
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return int(b.Version() - a.Version())
	})

	return m.applyMigrations(ctx, migrations, DirectionDown, conn, opts)
}

// applyMigrations executes the given migrations in the specified direction.]
//
// It assumes the passed migrations are already sorted in the necessary order.
func (m *Migrator) applyMigrations(
	ctx context.Context,
	migrations []*conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (result *MigrateResult, err error) {
	if opts.Steps != allSteps {
		migrations = migrations[0:min(opts.Steps, len(migrations))]
	}

	results := make([]MigrationResult, len(migrations))
	for i, migration := range migrations {
		var tx pgx.Tx
		inTx := must.Must(migration.UseTx(dir))

		m.logger.Debug(
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
					"conduit: failed to open transaction: %w",
					err,
				)
			}
			defer tx.Rollback(ctx)
		}

		start := time.Now()
		if err := migration.Apply(ctx, dir, conn, tx); err != nil {
			return nil, fmt.Errorf(
				"conduit: failed to apply migration %d: %w",
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
			return nil, fmt.Errorf("conduit: failed to update migrations table %v: %w", dir, err)
		}

		_ = dbsqlc.New().ResetConn(ctx, conn)

		if inTx {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf(
					"conduit: failed to commit transaction for migration %d: %w",
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

// pgLockNum computes a lock number for a PostgreSQL advisory lock.
//
// The input string is typically a registry namespace.
func pgLockNum(s string) int64 {
	h := fnv.New64()
	h.Write([]byte(s))
	n := int64(h.Sum64())

	d("generated advisory lock id: %d", n)

	return n
}
