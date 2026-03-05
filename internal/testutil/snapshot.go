package testutil

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// SnapshotFS recursively reads all files from the given afero filesystem
// directory and returns a sorted string representation suitable for
// snapshotting. File paths are relative to dir.
func SnapshotFS(t *testing.T, afs afero.Fs, dir string) {
	t.Helper()

	var b bytes.Buffer

	require.NoError(t, afero.Walk(afs, dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		content, err := afero.ReadFile(afs, path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		b.WriteString("### ")
		b.WriteString(rel)
		b.WriteString(" ###\n")
		b.Write(content)
		b.WriteString("\n")

		return nil
	}))

	snaps.MatchSnapshot(t, b.String())
}
