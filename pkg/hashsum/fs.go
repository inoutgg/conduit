package hashsum

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

type fsStore struct {
	fs       afero.Fs
	filename string
}

func NewFSStore(fs afero.Fs, filename string) Store {
	return &fsStore{fs: fs, filename: filename}
}

func (s *fsStore) Compare(path string, existing []byte) (bool, []byte, error) {
	b, err := afero.ReadFile(s.fs, filepath.Join(path, s.filename))
	if err != nil {
		return false, nil, fmt.Errorf("failed to hash from %s: %w", s.filename, err)
	}

	actual := bytes.TrimSpace(b)
	if len(actual) == 0 {
		return false, nil, errors.New("hash is empty")
	}

	isEq := bytes.Equal(actual, existing)
	if !isEq {
		return false, actual, nil
	}

	return isEq, nil, nil
}

func (s *fsStore) Save(path string, hash []byte) error {
	if err := afero.WriteFile(s.fs, filepath.Join(path, s.filename), hash, 0o644); err != nil {
		return fmt.Errorf("failed to persist hash to %s: %w", s.filename, err)
	}

	return nil
}
