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
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/foundations/debug"
)

var _ Migrator = (*migrator)(nil)
var _ MigratorTx = (*migrator)(nil)

var (
	ErrNoConn = errors.New("conduit: no migration specified")
)

// A concatenation of encoded "migrations" letters by the following mapping: a=1,b=2,...z=26
const lockNum = int64(13971812091514)

const (
	DirectionUp   Direction = "up"   // rollup
	DirectionDown           = "down" // rollback
)

// Direction denotes whether SQL migration should be rolled up, or rolled back.
type Direction string

type MigrateFunc = conduitregistry.MigrateFunc
type Migration = conduitregistry.Migration

type Migrator interface {
	Migrate(context.Context, Direction) (*MigrateResult, error)
}

type MigratorTx interface {
	MigrateTx(context.Context, Direction, pgx.Tx) (*MigrateResult, error)
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
			Level: slog.LevelWarn,
		}))
	}

	c.Registry = cmp.Or(c.Registry, globalRegistry)
}

// MigrateResult is the result for a applied batch migrations.
type MigrateResult struct {
	Direction        Direction
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

type existingMigration struct {
	Version int64
	Name    string
}

func NewMigrator(conn *pgx.Conn, config *Config) Migrator {
	debug.Assert(conn == nil, "expected conn to be defined")
	debug.Assert(config.Logger == nil, "config.Logger must be defined")
	debug.Assert(config.Registry == nil, "config.Registry must be defined")

	return &migrator{
		conn:     conn,
		logger:   config.Logger,
		registry: config.Registry,
	}
}

// NewMigratorTx creates a new migration
func NewMigratorTx(config *Config) MigratorTx {
	debug.Assert(config.Logger == nil, "config.Logger must be defined")
	debug.Assert(config.Registry == nil, "config.Registry must be defined")

	return &migrator{
		conn:     nil,
		logger:   config.Logger,
		registry: config.Registry,
	}
}

type migrator struct {
	conn     *pgx.Conn
	logger   *slog.Logger
	registry *conduitregistry.Registry
}

// existingMigrationVerions returns a list of migration versions that
// have been already applied.
func (m *migrator) existingMigrationVerions(ctx context.Context) ([]int64, error) {
	existingMigrations, err := m.existingMigrations(ctx)
	if err != nil {
		return nil, err
	}

	version := make([]int64, len(existingMigrations))
	for i, existingMigration := range existingMigrations {
		version[i] = existingMigration.Version
	}

	return version, nil
}

func (m *migrator) existingMigrations(ctx context.Context) ([]*existingMigration, error) {
	existingMigrations, err := dbsqlc.New().FindAllExistingMigrations(
		ctx,
		m.conn,
		m.registry.Namespace,
	)
	if err != nil {
		return nil, err
	}

	return sliceutil.Map(existingMigrations, func(m dbsqlc.FindAllExistingMigrationsRow) *existingMigration {
		return &existingMigration{
			Version: m.Version,
			Name:    m.Name,
		}
	}), nil
}

func (m *migrator) Migrate(ctx context.Context, direction Direction) (*MigrateResult, error) {
	if err := dbsqlc.New().AcquireLock(ctx, m.conn, lockNum); err != nil {
		return nil, fmt.Errorf("conduit: unable to acquire a lock: %w", err)
	}
	defer dbsqlc.New().ReleaseLock(ctx, m.conn, lockNum)

	tx, err := m.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("conduit: unable to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	return m.MigrateTx(ctx, direction, tx)
}

func (m *migrator) MigrateTx(ctx context.Context, direction Direction, tx pgx.Tx) (result *MigrateResult, err error) {
	switch direction {
	case DirectionUp:
		result, err = m.migrateUp(ctx, tx)
	case DirectionDown:
		result, err = m.migrateDown(ctx, tx)
	default:
		return nil, errors.New("conduit: unknown direction, expected either up or down")
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *migrator) migrateUp(ctx context.Context, tx pgx.Tx) (*MigrateResult, error) {
	existingMigrations, err := m.existingMigrations(ctx)
	if err != nil {
		return nil, err
	}

	targetMigrations := m.registry.Migrations()
	for _, m := range existingMigrations {
		delete(targetMigrations, m.Version)
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b Migration) int {
		return int(a.Version() - b.Version())
	})

	result, err := m.applyMigrations(ctx, migrations, func(
		ctx context.Context,
		migration conduitregistry.Migration,
		tx pgx.Tx,
	) error {
		return migration.Down(ctx, tx)
	}, tx)
	if err != nil {
		return nil, err
	}

	rows := make([]dbsqlc.ApplyMigrationParams, len(result.MigrationResults))
	for i, r := range result.MigrationResults {
		rows[i] = dbsqlc.ApplyMigrationParams{
			Name:      r.Name,
			Version:   r.Version,
			Namespace: m.registry.Namespace,
		}
	}

	if _, err := dbsqlc.New().ApplyMigration(ctx, tx, rows); err != nil {
		return nil, fmt.Errorf("conduit: an error occurred while updating migrations table: %w", err)
	}

	return result, nil
}

func (m *migrator) migrateDown(ctx context.Context, tx pgx.Tx) (*MigrateResult, error) {
	existingMigrations, err := m.existingMigrationVerions(ctx)
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
	// last applied migration to the very first one.
	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b Migration) int {
		return int(b.Version() - a.Version())
	})

	result, err := m.applyMigrations(ctx, migrations, func(
		ctx context.Context,
		migration conduitregistry.Migration,
		tx pgx.Tx,
	) error {
		return migration.Down(ctx, tx)
	}, tx)
	if err != nil {
		return nil, err
	}

	if err := dbsqlc.New().RollbackMigrations(ctx, tx, dbsqlc.RollbackMigrationsParams{
		Namespaces: sliceutil.Map(result.MigrationResults, func(r MigrationResult) string { return r.Namespace }),
		Versions:   sliceutil.Map(result.MigrationResults, func(r MigrationResult) int64 { return r.Version }),
	}); err != nil {
		return nil, fmt.Errorf("conduit: an error occurred while updating migrations table: %w", err)
	}

	return result, err
}

func (m *migrator) applyMigrations(
	ctx context.Context,
	migrations []conduitregistry.Migration,
	apply func(context.Context, conduitregistry.Migration, pgx.Tx) error,
	tx pgx.Tx,
) (*MigrateResult, error) {
	results := make([]MigrationResult, len(migrations))
	for i, migration := range migrations {
		start := time.Now()
		if err := apply(ctx, migration, tx); err != nil {
			return nil, fmt.Errorf("conduit: an error occurred while applying migration %d: %w", migration.Version(), err)
		}

		duration := time.Since(start)
		results[i] = MigrationResult{
			DurationTotal: duration,
			Version:       migration.Version(),
			Name:          migration.Name(),
			Namespace:     m.registry.Namespace,
		}
	}

	return &MigrateResult{
		MigrationResults: results,
		Direction:        DirectionDown,
	}, nil
}
