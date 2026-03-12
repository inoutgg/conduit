package conduit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stripe/pg-schema-diff/pkg/schema"
	"go.inout.gg/foundations/debug"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/direction"
	"go.inout.gg/conduit/internal/internaldebug"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/stopwatch"
)

// AllSteps tells the migrator to apply every available migration.
const AllSteps = -1

const (
	DefaultUpStep   = AllSteps // default for up: apply all pending
	DefaultDownStep = 1        // default for down: roll back one
)

const (
	DirectionUp   Direction = direction.DirectionUp   // roll up
	DirectionDown           = direction.DirectionDown // roll down
)

var (
	ErrInvalidStep = errors.New(
		"invalid migration step: expected -1 (all) or positive integer",
	)
	ErrSchemaDrift    = errors.New("schema drift detected")
	ErrHazardDetected = errors.New("hazardous migration detected")
)

type (
	Direction = direction.Direction
	Migration = conduitregistry.Migration
)

type config struct {
	Logger               *slog.Logger
	Registry             *conduitregistry.Registry
	Executor             MigrationExecutor
	SkipSchemaDriftCheck bool
}

// Option configures a Migrator.
type Option func(*config)

// WithLogger sets the logger used by the Migrator for debug output.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) { c.Logger = l }
}

// WithRegistry sets the migration registry. When omitted, the global
// registry populated by [FromFS] is used.
func WithRegistry(r *conduitregistry.Registry) Option {
	return func(c *config) { c.Registry = r }
}

// WithExecutor sets the migration executor. When omitted, a live executor
// that applies migrations to the database is used.
func WithExecutor(e MigrationExecutor) Option {
	return func(c *config) { c.Executor = e }
}

// WithSkipSchemaDriftCheck disables the schema drift check that runs before
// applying up migrations.
func WithSkipSchemaDriftCheck() Option {
	return func(c *config) { c.SkipSchemaDriftCheck = true }
}

func (c *config) defaults() {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}

	if c.Registry == nil {
		c.Registry = globalRegistry
	}

	if c.Executor == nil {
		c.Executor = NewLiveExecutor(c.Logger, stopwatch.Standard{})
	}
}

// MigrationResult holds the outcome of a single applied migration.
type MigrationResult struct {
	Version       conduitversion.Version
	Name          string
	DurationTotal time.Duration
}

// MigrateOptions configures a single [Migrator.Migrate] call.
//
// Steps controls how many migrations to apply. Use [AllSteps] (-1) to apply
// all pending migrations. When zero, the direction-specific default is used
// ([DefaultUpStep] for up, [DefaultDownStep] for down).
//
// AllowHazards lists hazard types that are permitted to proceed. Migrations
// containing unlisted hazards cause [ErrHazardDetected].
type MigrateOptions struct {
	AllowHazards []HazardType
	Steps        int
}

func (m *MigrateOptions) defaults(dir direction.Direction) {
	if m.Steps == 0 {
		if dir == direction.DirectionUp {
			m.Steps = DefaultUpStep
		} else {
			m.Steps = DefaultDownStep
		}
	}
}

// Migrator rolls migrations up and down in version order.
type Migrator struct {
	logger               *slog.Logger
	registry             *conduitregistry.Registry
	executor             MigrationExecutor
	skipSchemaDriftCheck bool
}

// NewMigrator creates a Migrator configured with the given options.
func NewMigrator(opts ...Option) *Migrator {
	//nolint:exhaustruct
	cfg := &config{}
	for _, c := range opts {
		c(cfg)
	}

	cfg.defaults()

	debug.Assert(cfg.Logger != nil, "config.Logger must be defined")
	debug.Assert(cfg.Registry != nil, "config.Registry must be defined")

	return &Migrator{
		logger:               cfg.Logger,
		registry:             cfg.Registry,
		executor:             cfg.Executor,
		skipSchemaDriftCheck: cfg.SkipSchemaDriftCheck,
	}
}

// Migrate applies migrations in the given direction. It acquires a Postgres
// advisory lock for the duration of the operation.
//
// The returned iterator yields individual [MigrationResult] values as each
// migration completes. If a migration fails, the iterator yields the error
// and stops. The advisory lock is held for the duration of the iteration.
//
// When opts is nil, direction-specific defaults are used: all pending
// migrations for up, one migration for down.
func (m *Migrator) Migrate(
	ctx context.Context,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (iter.Seq2[*MigrationResult, error], error) {
	debug.Assert(conn != nil, "expected conn to be defined")

	if opts == nil {
		opts = new(MigrateOptions)

		internaldebug.Log("opts is omitted, create a new one")
	}

	opts.defaults(dir)

	debug.Assert(opts.Steps == -1 || opts.Steps > 0, "invalid steps")

	lockNum := pgLockNum("conduit")

	if err := dbsqlc.New().AcquireLock(ctx, conn, lockNum); err != nil {
		return nil, fmt.Errorf("failed to acquire a lock: %w", err)
	}

	defer func() { _ = dbsqlc.New().ReleaseLock(ctx, conn, lockNum) }()

	var (
		migrations []*conduitregistry.Migration
		err        error
	)

	switch dir {
	case DirectionUp:
		migrations, err = m.upMigrations(ctx, conn)
	case DirectionDown:
		migrations, err = m.downMigrations(ctx, conn)
	default:
		err = direction.ErrUnknownDirection
	}

	if err != nil {
		return nil, err
	}

	return m.applyMigrations(ctx, migrations, dir, conn, opts), nil
}

func (m *Migrator) existingMigrationKeys(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	ok, err := dbsqlc.New().DoesTableExist(ctx, conn, "conduit_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from migrations table: %w", err)
	}

	if !ok {
		internaldebug.Log("conduitmigrations table is not found")
		return []string{}, nil
	}

	rows, err := dbsqlc.New().AllExistingMigrations(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing migrations: %w", err)
	}

	keys := make([]string, len(rows))
	for i, row := range rows {
		keys[i] = row.Version + "_" + row.Name
	}

	return keys, nil
}

func (m *Migrator) upMigrations(
	ctx context.Context,
	conn *pgx.Conn,
) ([]*conduitregistry.Migration, error) {
	existingKeys, err := m.existingMigrationKeys(ctx, conn)
	if err != nil {
		return nil, err
	}

	if !m.skipSchemaDriftCheck {
		if err := m.detectSchemaDrift(ctx, conn); err != nil {
			return nil, err
		}
	}

	targetMigrations := m.registry.Migrations()
	for _, key := range existingKeys {
		delete(targetMigrations, key)
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, compareMigrations)

	return migrations, nil
}

func (m *Migrator) downMigrations(
	ctx context.Context,
	conn *pgx.Conn,
) ([]*conduitregistry.Migration, error) {
	existingKeys, err := m.existingMigrationKeys(ctx, conn)
	if err != nil {
		return nil, err
	}

	existingKeysSet := make(map[string]struct{}, len(existingKeys))
	for _, key := range existingKeys {
		existingKeysSet[key] = struct{}{}
	}

	targetMigrations := m.registry.Migrations()
	for key := range targetMigrations {
		if _, ok := existingKeysSet[key]; !ok {
			delete(targetMigrations, key)
		}
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return compareMigrations(b, a)
	})

	return migrations, nil
}

func (m *Migrator) detectSchemaDrift(ctx context.Context, conn *pgx.Conn) error {
	internaldebug.Log("detecting schema drift")

	expected, err := dbsqlc.New().LatestSchemaHash(ctx, conn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("failed to fetch latest schema hash: %w", err)
	}

	db := stdlib.OpenDB(*conn.Config())
	defer db.Close()

	actual, err := schema.GetSchemaHash(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to compute schema hash: %w", err)
	}

	if actual != expected {
		return fmt.Errorf(
			"%w: expected hash %s, got %s",
			ErrSchemaDrift,
			expected,
			actual,
		)
	}

	return nil
}

func (m *Migrator) applyMigrations(
	ctx context.Context,
	migrations []*conduitregistry.Migration,
	dir Direction,
	conn *pgx.Conn,
	opts *MigrateOptions,
) iter.Seq2[*MigrationResult, error] {
	return func(yield func(*MigrationResult, error) bool) {
		if opts.Steps != AllSteps {
			migrations = migrations[0:min(opts.Steps, len(migrations))]
		}

		internaldebug.Log(
			"running migrations migrations=[%s] steps=%d total_migrations_count=%d",
			strings.Join(sliceutil.Map(migrations, func(m *conduitregistry.Migration) string {
				return fmt.Sprintf("name=%s version=%s", m.Name(), m.Version().String())
			}), ", "),
			opts.Steps,
			len(migrations),
		)

		for _, migration := range migrations {
			internaldebug.Log(
				"running migration name=%s version=%s direction=%s",
				migration.Name(),
				migration.Version().String(),
				dir,
			)

			if hazards := migration.Hazards(dir); len(hazards) > 0 {
				var blocked []string

				for _, h := range hazards {
					if !slices.Contains(opts.AllowHazards, h.Type) {
						blocked = append(blocked, fmt.Sprintf("%s: %s", h.Type, h.Message))
					}
				}

				if len(blocked) > 0 {
					yield(nil, fmt.Errorf(
						"%w: migration %s_%s contains hazards:\n  - %s",
						ErrHazardDetected,
						migration.Version().String(),
						migration.Name(),
						strings.Join(blocked, "\n  - "),
					))

					return
				}
			}

			migrationResult, err := m.executor.Execute(ctx, migration, dir, conn)
			if err != nil {
				yield(nil, err)

				return
			}

			if !yield(&migrationResult, nil) {
				return
			}
		}
	}
}

func compareMigrations(a, b *conduitregistry.Migration) int {
	if c := a.Version().Compare(b.Version()); c != 0 {
		return c
	}

	return cmp.Compare(a.Name(), b.Name())
}

func pgLockNum(s string) int64 {
	h := fnv.New32a()
	h.Write([]byte(s))
	n := h.Sum32()

	internaldebug.Log("generated advisory lock id: %d", n)

	return int64(n)
}
