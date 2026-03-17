// Package lockfile provides an interface for reading and persisting
// per-migration schema hashes used for chain-level drift detection.
package lockfile

import "go.inout.gg/conduit/pkg/conduitversion"

// Entry represents a single migration and its cumulative schema hash.
type Entry struct {
	Parsed conduitversion.ParsedMigrationFilename
	Hash   string
}

// Store reads and persists lockfile entries.
type Store interface {
	// Read returns all entries from the lockfile at path.
	// Returns an empty slice and no error when the lockfile does not exist.
	Read(path string) ([]Entry, error)

	// Save writes entries to the lockfile at path.
	Save(path string, entries []Entry) error
}
