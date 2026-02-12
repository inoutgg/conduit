package testutil

import (
	"bytes"
	"path"
	"sort"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// SnapshotFS reads all files from the given afero filesystem directory
// and returns a sorted string representation suitable for snapshotting.
func SnapshotFS(t *testing.T, fs afero.Fs, dir string) {
	t.Helper()

	var b bytes.Buffer

	entries, err := afero.ReadDir(fs, dir)
	require.NoError(t, err)

	// Sort entries by name for deterministic output.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := afero.ReadFile(fs, path.Join(dir, entry.Name()))
		require.NoError(t, err)

		b.WriteString("### ")
		b.WriteString(entry.Name())
		b.WriteString(" ###\n")
		b.Write(content)
		b.WriteString("\n")
	}

	snaps.MatchSnapshot(t, b.String())
}
