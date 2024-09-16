package version

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// MigrationFilename returns a filename of the desired migration file.
func MigrationFilename(version int64, name string, ext string) string {
	return fmt.Sprintf("%d_%s.%s", version, name, ext)
}

// ParsedMigrationFilename represents a result of parsing migration filename.
type ParsedMigrationFilename struct {
	// Unix milliseconds
	Version int64

	// Human-readable part of filename
	Name string

	// Extension of the migration file. Either "sql" or "go"
	Extension string
}

// ParseMigrationFilename parses a filename of `<numeric-version-part>_<string-part>.[go|sql]`
// format.
// Example: 1257894000000_create_user.sql, 1257894454320_create_projects.go
// 1257894000000_create_user.sql -> 1257894000000, create_user, sql
func ParseMigrationFilename(filename string) (*ParsedMigrationFilename, error) {
	basename := filepath.Base(filename)
	if basename == "." {
		return nil, fmt.Errorf("conduit: filename cannot be empty")
	}

	ext := filepath.Ext(basename)
	if ext != ".go" && ext != ".sql" {
		return nil, fmt.Errorf("conduit: unknown migration file extension \"%s\", expected: \".sql\" or \".go\"", ext)
	}

	stringVersion, name, ok := strings.Cut(basename[:len(basename)-len(ext)], "_")
	if !ok {
		return nil, fmt.Errorf("conduit: malformed migration filename, expected format: <numeric-version-part>_<string-part>.[go|sql], was: %s", basename)
	}

	version, err := parseVersionString(stringVersion)
	if err != nil {
		return nil, err
	}

	return &ParsedMigrationFilename{
		Version: version,
		Name:    name,

		// Drop dot from extension
		Extension: ext[1:],
	}, nil
}

// parseVersionString parses version of the migration into a unix time.
func parseVersionString(version string) (int64, error) {
	numericVersion, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("conduit: unable parse version \"%s\": %w", version, err)
	}

	if numericVersion < 0 {
		return 0, fmt.Errorf("conduit: invalid version")
	}

	return numericVersion, nil
}
