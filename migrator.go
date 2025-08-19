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
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/direction"
	internaldebug "go.inout.gg/conduit/internal/internaldebug"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/internal/uuidv7"
	"go.inout.gg/conduit/internal/version"
)

// AllSteps tells migrator to run all available migrations either up or down.
const AllSteps = -1

const (
	DefaultUpStep   = AllSteps // roll up
	DefaultDownStep = 1        // roll back
)

const (
	DirectionUp   Direction = direction.DirectionUp   // roll up
	DirectionDown           = direction.DirectionDown // roll down
)

var ErrInvalidStep = errors.New(
	"conduit: invalid migration step. Expected: -1 (all) or positive integer",
)

type (
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
// If Config.Logger is not provided, it falls back to slog.Default.
func NewConfig(cfgs ...func(*Config)) *Config {
	//nolint:exhaustruct
	config := &Config{}
	for _, c := range cfgs {
		c(config)
	}

	config.defaults()

	debug.Assert(config.Logger != nil, "Logger is required")
	debug.Assert(config.Registry != nil, "Registry is required")

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
	Version       version.Version
	Name          string
	Namespace     string
	DurationTotal time.Duration
}

// MigrateOptions specifies options for a Migrator.Migrate operation.
type MigrateOptions struct {
	Steps int
}

func (m *MigrateOptions) validate() error {
	if m.Steps != -1 && m.Steps <= 0 {
		return ErrInvalidStep
	}

	return nil
}

// Migrator is a database migration tool that can rolls up and down migrations
// in order.
type Migrator struct {
	logger   *slog.Logger
	registry *conduitregistry.Registry
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
//
//nolint:nonamedreturns
func (m *Migrator) Migrate(
	ctx context.Context,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (result *MigrateResult, err error) {
	debug.Assert(conn != nil, "expected conn to be defined")

	if opts == nil {
		opts = &MigrateOptions{Steps: DefaultUpStep}
		if dir == DirectionDown {
			opts.Steps = DefaultDownStep
		}

		internaldebug.Log("opts is omitted, using the default one: %v", opts)
	}

	if err := opts.validate(); err != nil {
		return nil, err
	}

	lockNum := pgLockNum(m.registry.Namespace)

	if err := dbsqlc.New().AcquireLock(ctx, conn, lockNum); err != nil {
		return nil, fmt.Errorf("conduit: failed to acquire a lock: %w", err)
	}

	defer func() {
		_ = dbsqlc.New().ReleaseLock(ctx, conn, lockNum)
	}()

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

// existingMigrationVersions retrieves a list of already applied migration versions.
func (m *Migrator) existingMigrationVersions(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	ok, err := dbsqlc.New().DoesTableExist(ctx, conn, "conduitmigrations")
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch from migrations table: %w", err)
	}

	if !ok {
		internaldebug.Log("conduitmigrations table is not found")
		return []string{}, nil
	}

	versions, err := dbsqlc.New().AllExistingMigrationVersions(ctx, conn, m.registry.Namespace)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch existing versions: %w", err)
	}

	return versions, nil
}

// migrateUp applies pending migrations in the up direction.
//
// Migrations are rolled up in ascending order.
func (m *Migrator) migrateUp(
	ctx context.Context,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (*MigrateResult, error) {
	existingMigrationVersions, err := m.existingMigrationVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	targetMigrations := m.registry.CloneMigrations()
	for _, existingVersion := range existingMigrationVersions {
		delete(targetMigrations, existingVersion)
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return a.Version().Compare(b.Version())
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
	existingMigrations, err := m.existingMigrationVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	// Filter only applied migrations.
	existingMigrationsMap := sliceutil.KeyBy(existingMigrations, func(e string) string { return e })
	targetMigrations := m.registry.CloneMigrations()

	for _, m := range targetMigrations {
		if _, ok := existingMigrationsMap[m.Version().String()]; !ok {
			delete(targetMigrations, m.Version().String())
		}
	}

	// Sort in descending order, as we need to roll back starting from the
	// last applied migration to the first one.
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return b.Version().Compare(a.Version())
	})

	return m.applyMigrations(ctx, migrations, DirectionDown, conn, opts)
}

// applyMigrations executes the given migrations in the specified direction.
//
// It assumes the passed migrations are already sorted in the necessary order.
//
//nolint:nonamedreturns
func (m *Migrator) applyMigrations(
	ctx context.Context,
	migrations []*conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (result *MigrateResult, err error) {
	if opts.Steps != AllSteps {
		migrations = migrations[0:min(opts.Steps, len(migrations))]
	}

	results := make([]MigrationResult, len(migrations))

	internaldebug.Log(
		"running migrations migrations=[%s] steps=%d total_migrations_count=%d",
		strings.Join(sliceutil.Map(migrations, func(m *conduitregistry.Migration) string {
			return fmt.Sprintf("name=%s version=%s", m.Name(), m.Version().String())
		}), ", "),
		opts.Steps,
		len(migrations),
	)

	for i, migration := range migrations {
		internaldebug.Log(
			"running migration name=%s version=%s direction=%s",
			migration.Name(),
			migration.Version().String(),
			dir,
		)

		var tx pgx.Tx

		inTx := must.Must(migration.UseTx(dir))

		m.logger.DebugContext(
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

		if inTx {
			tx, err = conn.Begin(ctx)
			if err != nil {
				return nil, fmt.Errorf(
					"conduit: failed to open transaction: %w",
					err,
				)
			}

			defer func() { _ = tx.Rollback(ctx) }()
		}

		start := time.Now()

		err := migration.Apply(ctx, dir, conn, tx)
		if err != nil {
			return nil, fmt.Errorf(
				"conduit: failed to apply migration %s: %w",
				migration.Version().String(),
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
				Version:   migrationResult.Version.String(),
				Namespace: migrationResult.Namespace,
			})

		case DirectionUp:
			err = dbsqlc.New().ApplyMigration(ctx, conn, dbsqlc.ApplyMigrationParams{
				ID:        uuidv7.Must(),
				Version:   migrationResult.Version.String(),
				Namespace: migrationResult.Namespace,
				Name:      migrationResult.Name,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("conduit: failed to update migrations table %v: %w", dir, err)
		}

		if inTx {
			err := tx.Commit(ctx)
			if err != nil {
				return nil, fmt.Errorf(
					"conduit: failed to commit transaction for migration %s: %w",
					migrationResult.Version.String(),
					err,
				)
			}
		}

		_ = dbsqlc.New().ResetConn(ctx, conn)
	}

	result = &MigrateResult{
		MigrationResults: results,
		Direction:        DirectionDown,
	}

	return result, nil
}

// pgLockNum computes a lock number for a PostgreSQL advisory lock.
//
// The input string is typically a registry namespace.
func pgLockNum(s string) int64 {
	h := fnv.New32a()
	h.Write([]byte(s))
	n := h.Sum32()

	internaldebug.Log("generated advisory lock id: %d", n)

	return int64(n)
}
