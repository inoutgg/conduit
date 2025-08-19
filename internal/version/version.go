package version

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const format = "20060102150405" // YYYYMMDDHHMMSS

type Version struct{ t time.Time }

func NewVersion() Version             { return Version{t: time.Now()} }
func NewFromTime(t time.Time) Version { return Version{t: t} }

// String returns the version as a string.
func (v Version) String() string { return v.t.Format(format) }

// Compare compares current version and the other one,
// returning -1 if current version is older, 0 if versions are equal,
// and 1 if current version is newer.
func (v Version) Compare(other Version) int { return v.t.Compare(other.t) }

// MigrationFilename generates a filename for a migration file.
func MigrationFilename(v Version, name string, ext string) string {
	return fmt.Sprintf("%s_%s.%s", v.String(), name, ext)
}

// ParsedMigrationFilename represents the components of a parsed migration filename.
type ParsedMigrationFilename struct {
	Version   Version
	Name      string
	Extension string
}

// Example: 1257894000000_create_user.sql -> 1257894000000, create_user, sql.
func ParseMigrationFilename(filename string) (*ParsedMigrationFilename, error) {
	basename := filepath.Base(filename)
	if basename == "." {
		return nil, errors.New("conduit: filename cannot be empty")
	}

	ext := filepath.Ext(basename)
	if ext != ".go" && ext != ".sql" {
		return nil, fmt.Errorf("conduit: unknown migration file extension %q, expected: .sql or .go", ext)
	}

	version, name, ok := strings.Cut(basename[:len(basename)-len(ext)], "_")
	if !ok {
		return nil, fmt.Errorf(
			"conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: %s",
			basename,
		)
	}

	ver, err := time.Parse(format, version)
	if err != nil {
		return nil, fmt.Errorf("conduit: invalid version format %q, expected: YYYYMMDDHHMMSS: %w", version, err)
	}

	return &ParsedMigrationFilename{
		Version:   Version{ver},
		Name:      name,
		Extension: ext[1:], // Drop leading dot from extension
	}, nil
}
