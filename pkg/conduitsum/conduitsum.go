package conduitsum

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spf13/afero"
)

// Filename is the name of the checksum file that tracks the expected schema hash
// in a migrations directory.
const Filename = "conduit.sum"

// ReadFile reads and parses a conduit.sum file from the given filesystem.
func ReadFile(fs afero.Fs) (string, error) {
	data, err := afero.ReadFile(fs, Filename)
	if err != nil {
		return "", fmt.Errorf("failed to read conduit.sum: %w", err)
	}

	hash := bytes.TrimSpace(data)

	if len(hash) == 0 {
		return "", errors.New("conduit.sum file is empty")
	}

	return string(hash), nil
}

// WriteFile writes a schema hash to a conduit.sum file in the given filesystem.
func WriteFile(fs afero.Fs, hash string) error {
	if err := afero.WriteFile(fs, Filename, []byte(hash+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write conduit.sum: %w", err)
	}

	return nil
}
