package conduitcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.inout.gg/conduit"
	"go.inout.gg/foundations/env"
)

var (
	DefaultMigrationDir = "migrations"
)

var _ Interface = (*cli)(nil)

// Inferface exposes public CLI interface.
type Interface interface {
	// Execute executes a command if matched.
	Execute(context.Context) error
}

type cli struct {
	cmd    *cobra.Command
	config *Config
}

type Config struct {
	MigrationDir string
}

// ConfigFromEnv loads configurations using .env file.
// If some options are missing it will fallback to default ones.
func ConfigFromEnv() (*Config, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("conduit: failed to load working directory: %w", err)
	}

	config, err := env.Load[Config](filepath.Clean(filepath.Join(path, ".env")))
	if err != nil {
		return nil, err
	}
	config.defaults()

	return config, nil
}

func (c *Config) defaults() {
	if c.MigrationDir == "" {
		c.MigrationDir = DefaultMigrationDir
	}
}

func New(migrator conduit.Migrator, config *Config) Interface {
	rootCmd := &cobra.Command{}

	rootCmd.AddCommand(
		newCommandInit(),
		newCommandCreate(),
		newCommandApply(migrator),
	)

	return &cli{
		cmd: rootCmd,
	}
}

func (c *cli) Execute(ctx context.Context) error { return c.cmd.ExecuteContext(ctx) }
