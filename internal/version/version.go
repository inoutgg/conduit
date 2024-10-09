package version

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// MigrationFilename generates a filename for a migration file.
func MigrationFilename(version int64, name string, ext string) string {
	return fmt.Sprintf("%d_%s.%s", version, name, ext)
}

// ParsedMigrationFilename represents the components of a parsed migration filename.
type ParsedMigrationFilename struct {
	Version   int64  // Unix milliseconds
	Name      string // Human-readable part of filename
	Extension string // File extension (either "sql" or "go")
}

// ParseMigrationFilename parses a filename of format "<version>_<name>.[go|sql]".
// Example: 1257894000000_create_user.sql -> 1257894000000, create_user, sql
func ParseMigrationFilename(filename string) (*ParsedMigrationFilename, error) {
	basename := filepath.Base(filename)
	if basename == "." {
		return nil, fmt.Errorf("conduit: filename cannot be empty")
	}

	ext := filepath.Ext(basename)
	if ext != ".go" && ext != ".sql" {
		return nil, fmt.Errorf("conduit: unknown migration file extension %q, expected: .sql or .go", ext)
	}

	stringVersion, name, ok := strings.Cut(basename[:len(basename)-len(ext)], "_")
	if !ok {
		return nil, fmt.Errorf("conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: %s", basename)
	}

	version, err := parseVersionString(stringVersion)
	if err != nil {
		return nil, err
	}

	return &ParsedMigrationFilename{
		Version:   version,
		Name:      name,
		Extension: ext[1:], // Drop leading dot from extension
	}, nil
}

// parseVersionString converts a version string to a Unix timestamp.
func parseVersionString(version string) (int64, error) {
	numericVersion, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("conduit: unable to parse version %q: %w", version, err)
	}

	if numericVersion < 0 {
		return 0, fmt.Errorf("conduit: invalid version")
	}

	return numericVersion, nil
}
