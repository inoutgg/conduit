// Package hashsum provides an interface for comparing and persisting schema
// hashes used for drift detection.
package hashsum

// Store compares and persists schema hashes.
type Store interface {
	// Compare checks whether existing matches the hash stored at path.
	// Returns (true, nil, nil) on match. On mismatch returns (false, actual, nil).
	Compare(path string, existing []byte) (bool, []byte, error)

	// Save writes hash to the store at path.
	Save(path string, hash []byte) error
}
