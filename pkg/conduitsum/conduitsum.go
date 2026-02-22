package conduitsum

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spf13/afero"
)

const Filename = "conduit.sum"

// ReadFile reads and parses a conduit.sum file from the given filesystem.
func ReadFile(fs afero.Fs) (string, error) {
	data, err := afero.ReadFile(fs, Filename)
	if err != nil {
		return "", fmt.Errorf("conduitsum: failed to read file: %w", err)
	}

	hash := bytes.TrimSpace(data)

	if len(hash) == 0 {
		return "", errors.New("conduitsum: empty file")
	}

	return string(hash), nil
}

// WriteFile writes a schema hash to a conduit.sum file in the given filesystem.
func WriteFile(fs afero.Fs, hash string) error {
	if err := afero.WriteFile(fs, Filename, []byte(hash+"\n"), 0o644); err != nil {
		return fmt.Errorf("conduitsum: failed to write file: %w", err)
	}

	return nil
}
