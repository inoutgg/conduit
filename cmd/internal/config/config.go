package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
	"sigs.k8s.io/yaml/kyaml"
)

const DefaultFilename = "conduit.yaml"

// FilePath resolves a file:// URL to its local filesystem path.
func FilePath(uri *url.URL) (string, error) {
	if uri == nil {
		return "", nil
	}

	if uri.Scheme != "file" {
		return "", fmt.Errorf("unsupported URL scheme %q (only file:// is supported)", uri.Scheme)
	}

	return uri.Path, nil
}

type DatabaseConfig struct {
	URL string `json:"url"`
}

type MigrationsConfig struct {
	Dir    *url.URL
	Schema *url.URL
}

func (m *MigrationsConfig) UnmarshalJSON(data []byte) error {
	var raw struct {
		Dir    string `json:"dir"`
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal migrations config: %w", err)
	}

	if raw.Dir != "" {
		u, err := url.Parse(raw.Dir)
		if err != nil {
			return fmt.Errorf("invalid dir URL %q: %w", raw.Dir, err)
		}

		m.Dir = u
	}

	if raw.Schema != "" {
		u, err := url.Parse(raw.Schema)
		if err != nil {
			return fmt.Errorf("invalid schema URL %q: %w", raw.Schema, err)
		}

		m.Schema = u
	}

	return nil
}

func (m MigrationsConfig) MarshalJSON() ([]byte, error) {
	var raw struct {
		Dir    string `json:"dir,omitempty"`
		Schema string `json:"schema,omitempty"`
	}
	if m.Dir != nil {
		raw.Dir = m.Dir.String()
	}

	if m.Schema != nil {
		raw.Schema = m.Schema.String()
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal migrations config: %w", err)
	}

	return b, nil
}

//nolint:tagliatelle
type ApplyConfig struct {
	AllowHazards         []string `json:"allow_hazards"`
	SkipSchemaDriftCheck bool     `json:"skip_schema_drift_check"`
}

type Config struct {
	Migrations MigrationsConfig `json:"migrations"`
	Database   DatabaseConfig   `json:"database"`
	Apply      ApplyConfig      `json:"apply"`
	Verbose    bool             `json:"verbose"`
}

func (cfg *Config) validate() error {
	if _, err := FilePath(cfg.Migrations.Dir); err != nil {
		return fmt.Errorf("migrations.dir: %w", err)
	}

	if _, err := FilePath(cfg.Migrations.Schema); err != nil {
		return fmt.Errorf("migrations.schema: %w", err)
	}

	return nil
}

func WriteFile(fs afero.Fs, path string, cfg Config) error {
	data, err := new(kyaml.Encoder).Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := afero.WriteFile(fs, path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file %q: %w", path, err)
	}

	return nil
}

func FromFS(fs afero.Fs, path string) (Config, error) {
	var config Config

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}

		return config, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	if err := config.validate(); err != nil {
		return config, fmt.Errorf("invalid config file %q: %w", path, err)
	}

	return config, nil
}
