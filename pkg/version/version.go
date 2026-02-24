// Package version provides utilities for working with migration file versions.
package version

import (
	"cmp"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const format = "20060102150405" // YYYYMMDDHHMMSS

type Version struct{ t time.Time }

func NewFromTime(t time.Time) Version { return Version{t: t} }

func (v Version) String() string { return v.t.Format(format) }

// Compare compares current version and the other one,
// returning -1 if current version is older, 0 if versions are equal,
// and 1 if current version is newer.
func (v Version) Compare(other Version) int { return v.t.Compare(other.t) }

// MigrationDirection indicates whether a migration file is up-only or down-only.
type MigrationDirection string

const (
	// MigrationDirectionUp indicates an up-only migration file (.up.sql).
	MigrationDirectionUp MigrationDirection = "up"
	// MigrationDirectionDown indicates a down-only migration file (.down.sql).
	MigrationDirectionDown MigrationDirection = "down"
)

// MigrationFilename generates a filename for a SQL migration file.
func MigrationFilename(v Version, name string, direction MigrationDirection) string {
	return fmt.Sprintf("%s_%s.%s.sql", v.String(), name, direction)
}

// ParsedMigrationFilename represents the components of a parsed migration filename.
type ParsedMigrationFilename struct {
	Version   Version
	Name      string
	Direction MigrationDirection
}

// Compare compares current ParsedMigrationFilename and the other one.
//
// It compares by Version first, then by Name lexicographically when
// versions are equal (e.g. split migrations with the same timestamp).
func (f ParsedMigrationFilename) Compare(other ParsedMigrationFilename) int {
	if c := f.Version.Compare(other.Version); c != 0 {
		return c
	}

	return cmp.Compare(f.Name, other.Name)
}

func (f ParsedMigrationFilename) Filename() string {
	return MigrationFilename(f.Version, f.Name, f.Direction)
}

// ParseMigrationFilename parses a migration filename into its components.
//
// Supported formats:
//   - <version>_<name>.up.sql — up migration
//   - <version>_<name>.down.sql — down migration
func ParseMigrationFilename(filename string) (ParsedMigrationFilename, error) {
	var m ParsedMigrationFilename

	basename := filepath.Base(filename)
	if basename == "." {
		return m, errors.New("conduit: filename cannot be empty")
	}

	ext := filepath.Ext(basename)
	if ext != ".sql" {
		return m, fmt.Errorf(
			"conduit: unknown migration file extension %q, expected: .sql", ext)
	}

	// Check for direction suffix (.up.sql or .down.sql).
	withoutExt := strings.TrimSuffix(basename, ext)

	var direction MigrationDirection

	switch {
	case strings.HasSuffix(withoutExt, ".up"):
		direction = MigrationDirectionUp
		withoutExt = strings.TrimSuffix(withoutExt, ".up")
	case strings.HasSuffix(withoutExt, ".down"):
		direction = MigrationDirectionDown
		withoutExt = strings.TrimSuffix(withoutExt, ".down")
	default:
		return m, fmt.Errorf(
			"conduit: SQL migration file %q must have .up.sql or .down.sql suffix", basename)
	}

	version, name, ok := strings.Cut(withoutExt, "_")
	if !ok {
		return m, fmt.Errorf(
			"conduit: malformed migration filename, expected: <version>_<name>.sql, got: %s",
			basename,
		)
	}

	ver, err := time.Parse(format, version)
	if err != nil {
		return m, fmt.Errorf(
			"conduit: invalid version format %q, expected: YYYYMMDDHHMMSS: %w", version, err)
	}

	m = ParsedMigrationFilename{
		Version:   Version{ver},
		Name:      name,
		Direction: direction,
	}

	return m, nil
}
