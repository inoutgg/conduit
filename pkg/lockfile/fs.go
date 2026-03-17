package lockfile

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/pkg/conduitversion"
)

const header = "# conduit.lock — do not edit manually\n"

type fsStore struct {
	fs       afero.Fs
	filename string
}

func NewFSStore(fs afero.Fs, filename string) Store {
	return &fsStore{fs: fs, filename: filename}
}

func (s *fsStore) Read(path string) ([]Entry, error) {
	b, err := afero.ReadFile(s.fs, filepath.Join(path, s.filename))
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read %s: %w", s.filename, err)
	}

	var entries []Entry

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, hash, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("malformed lockfile entry: %s", line)
		}

		parsed, err := conduitversion.ParseMigrationFilename(key + ".up.sql")
		if err != nil {
			return nil, fmt.Errorf("malformed lockfile key %q: %w", key, err)
		}

		entries = append(entries, Entry{
			Parsed: parsed,
			Hash:   strings.TrimSpace(hash),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", s.filename, err)
	}

	return entries, nil
}

func (s *fsStore) Save(path string, entries []Entry) error {
	var buf bytes.Buffer
	buf.WriteString(header)
	buf.WriteByte('\n')

	for _, e := range entries {
		fmt.Fprintf(&buf, "%s %s\n", e.Parsed.String(), e.Hash)
	}

	if err := afero.WriteFile(s.fs, filepath.Join(path, s.filename), buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to persist %s: %w", s.filename, err)
	}

	return nil
}
