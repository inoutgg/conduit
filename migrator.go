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
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stripe/pg-schema-diff/pkg/schema"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/must"

	"go.inout.gg/conduit/conduitregistry"
	"go.inout.gg/conduit/internal/dbsqlc"
	"go.inout.gg/conduit/internal/direction"
	internaldebug "go.inout.gg/conduit/internal/internaldebug"
	"go.inout.gg/conduit/internal/sliceutil"
	"go.inout.gg/conduit/pkg/version"
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

var (
	ErrInvalidStep = errors.New(
		"conduit: invalid migration step. Expected: -1 (all) or positive integer",
	)

	ErrSchemaDrift    = errors.New("conduit: schema drift detected")
	ErrHazardDetected = errors.New("conduit: hazardous migration detected")
)

type (
	Direction = direction.Direction
	Migration = conduitregistry.Migration
)

// Config is the configuration for the Migrator.
//
// Use NewConfig to instantiate a new instance.
//
// Logger and Registry fields are optional.
// If Registry is omitted, the global registry is used.
// If Logger is omitted, slog.Default is used.
type Config struct {
	Logger                 *slog.Logger              // optional
	Registry               *conduitregistry.Registry // optional
	ShouldCheckSchemaDrift bool                      // optional
	AllowHazards           bool                      // optional
}

// Option is a function that configures a Config.
type Option func(*Config)

// WithLogger adds a logger to the Config.
//
// It's provided for convenience and intended to be used with NewConfig.
func WithLogger(l *slog.Logger) Option {
	return func(c *Config) { c.Logger = l }
}

// WithRegistry adds a registry to the Config.
//
// It's provided for convenience and intended to be used with NewConfig.
func WithRegistry(r *conduitregistry.Registry) Option {
	return func(c *Config) { c.Registry = r }
}

// WithNoSchemaDriftCheck disables schema drift check.
//
// It's provided for convenience and intended to be used with NewConfig.
func WithNoSchemaDriftCheck() Option {
	return func(c *Config) { c.ShouldCheckSchemaDrift = false }
}

// WithAllowHazards allows applying migrations that contain hazardous operations.
//
// It's provided for convenience and intended to be used with NewConfig.
func WithAllowHazards() Option {
	return func(c *Config) { c.AllowHazards = true }
}

// NewConfig creates a new Config and applies the provided configurations.
//
// If Config.Registry is not provided, it falls back to the global registry.
// If Config.Logger is not provided, it falls back to slog.Default.
func NewConfig(opts ...Option) *Config {
	//nolint:exhaustruct
	config := &Config{}
	for _, c := range opts {
		c(config)
	}

	config.defaults()

	debug.Assert(config.Logger != nil, "Logger is required")
	debug.Assert(config.Registry != nil, "Registry is required")

	return config
}

func (c *Config) defaults() {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}

	c.Registry = cmp.Or(c.Registry, globalRegistry)
	c.ShouldCheckSchemaDrift = cmp.Or(c.ShouldCheckSchemaDrift, true)
}

// MigrateResult represents the outcome of applied migrations batch.
type MigrateResult struct {
	// Direction of the applied migrations.
	Direction direction.Direction

	// MigrationResults is the result of applied migrations.
	MigrationResults []MigrationResult
}

// MigrationResult represents the outcome of a single applied migration.
type MigrationResult struct {
	// Version is the version of the applied migration.
	Version       version.Version
	Name          string
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
	logger                 *slog.Logger
	registry               *conduitregistry.Registry
	shouldCheckSchemaDrift bool
	allowHazards           bool
}

// NewMigrator creates a new migrator with the given config.
func NewMigrator(config *Config) *Migrator {
	debug.Assert(config.Logger != nil, "config.Logger must be defined")
	debug.Assert(config.Registry != nil, "config.Registry must be defined")

	return &Migrator{
		logger:                 config.Logger,
		registry:               config.Registry,
		shouldCheckSchemaDrift: config.ShouldCheckSchemaDrift,
		allowHazards:           config.AllowHazards,
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

	lockNum := pgLockNum("conduit")

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

func (m *Migrator) existingMigrationKeys(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	ok, err := dbsqlc.New().DoesTableExist(ctx, conn, "conduit_migrations")
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch from migrations table: %w", err)
	}

	if !ok {
		internaldebug.Log("conduitmigrations table is not found")
		return []string{}, nil
	}

	rows, err := dbsqlc.New().AllExistingMigrations(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to fetch existing migrations: %w", err)
	}

	keys := make([]string, len(rows))
	for i, row := range rows {
		keys[i] = row.Version + "_" + row.Name
	}

	return keys, nil
}

func (m *Migrator) migrateUp(
	ctx context.Context,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (*MigrateResult, error) {
	existingKeys, err := m.existingMigrationKeys(ctx, conn)
	if err != nil {
		return nil, err
	}

	if m.shouldCheckSchemaDrift {
		if err := m.detectSchemaDrift(ctx, conn); err != nil {
			return nil, err
		}
	}

	targetMigrations := m.registry.CloneMigrations()
	for _, key := range existingKeys {
		delete(targetMigrations, key)
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, compareMigrations)

	return m.applyMigrations(ctx, migrations, DirectionUp, conn, opts)
}

func (m *Migrator) migrateDown(
	ctx context.Context,
	conn *pgx.Conn,
	opts *MigrateOptions,
) (*MigrateResult, error) {
	existingKeys, err := m.existingMigrationKeys(ctx, conn)
	if err != nil {
		return nil, err
	}

	existingKeysMap := sliceutil.KeyBy(existingKeys, func(e string) string { return e })
	targetMigrations := m.registry.CloneMigrations()

	for key := range targetMigrations {
		if _, ok := existingKeysMap[key]; !ok {
			delete(targetMigrations, key)
		}
	}

	migrations := slices.Collect(maps.Values(targetMigrations))
	slices.SortFunc(migrations, func(a, b *Migration) int {
		return compareMigrations(b, a)
	})

	return m.applyMigrations(ctx, migrations, DirectionDown, conn, opts)
}

func (m *Migrator) detectSchemaDrift(ctx context.Context, conn *pgx.Conn) error {
	internaldebug.Log("detecting schema drift")

	expected, err := dbsqlc.New().LatestSchemaHash(ctx, conn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("conduit: failed to fetch latest schema hash: %w", err)
	}

	db := stdlib.OpenDB(*conn.Config())
	defer db.Close()

	actual, err := schema.GetSchemaHash(ctx, db)
	if err != nil {
		return fmt.Errorf("conduit: failed to compute schema hash: %w", err)
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

		if hazards := migration.Hazards(dir); !m.allowHazards && len(hazards) > 0 {
			msgs := make([]string, 0, len(hazards))
			for _, h := range hazards {
				msgs = append(msgs, fmt.Sprintf("%s: %s", h.Type, h.Message))
			}

			return nil, fmt.Errorf(
				"%w: migration %s_%s contains hazards:\n  - %s",
				ErrHazardDetected,
				migration.Version().String(),
				migration.Name(),
				strings.Join(msgs, "\n  - "),
			)
		}

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
			err = m.applyMigrationTx(ctx, migration, dir, conn)
		} else {
			err = migration.Apply(ctx, dir, conn)
		}

		if err != nil {
			return nil, fmt.Errorf(
				"conduit: failed to apply migration %s: %w",
				migration.Version().String(),
				err,
			)
		}

		duration := time.Since(time.Now())
		migrationResult := MigrationResult{
			DurationTotal: duration,
			Version:       migration.Version(),
			Name:          migration.Name(),
		}
		results[i] = migrationResult

		switch dir {
		case DirectionDown:
			err = dbsqlc.New().RollbackMigration(ctx, conn, dbsqlc.RollbackMigrationParams{
				Version: migrationResult.Version.String(),
				Name:    migrationResult.Name,
			})

		case DirectionUp:
			var schemaHash string

			schemaHash, err = m.computeSchemaHash(ctx, conn)
			if err != nil {
				return nil, fmt.Errorf(
					"conduit: failed to compute schema hash after migration %s: %w",
					migration.Version().String(),
					err,
				)
			}

			err = dbsqlc.New().ApplyMigration(ctx, conn, dbsqlc.ApplyMigrationParams{
				Version: migrationResult.Version.String(),
				Name:    migrationResult.Name,
				Hash:    schemaHash,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("conduit: failed to update migrations table %v: %w", dir, err)
		}

		_ = dbsqlc.New().ResetConn(ctx, conn)
	}

	result = &MigrateResult{
		MigrationResults: results,
		Direction:        DirectionDown,
	}

	return result, nil
}

func (m *Migrator) computeSchemaHash(ctx context.Context, conn *pgx.Conn) (string, error) {
	db := stdlib.OpenDB(*conn.Config())
	defer db.Close()

	hash, err := schema.GetSchemaHash(ctx, db)
	if err != nil {
		return "", fmt.Errorf("conduit: failed to compute schema hash: %w", err)
	}

	return hash, nil
}

func (m *Migrator) applyMigrationTx(
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
