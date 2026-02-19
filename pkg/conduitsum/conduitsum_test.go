package conduitsum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("parses hashes", func(t *testing.T) {
		t.Parallel()

		data := []byte("abc123\ndef456\n")
		hashes, err := Parse(data)

		require.NoError(t, err)
		assert.Equal(t, []string{"abc123", "def456"}, hashes)
	})

	t.Run("skips empty lines", func(t *testing.T) {
		t.Parallel()

		data := []byte("\nabc123\n\n")
		hashes, err := Parse(data)

		require.NoError(t, err)
		assert.Equal(t, []string{"abc123"}, hashes)
	})

	t.Run("returns error on line with spaces", func(t *testing.T) {
		t.Parallel()

		data := []byte("abc 123\n")
		_, err := Parse(data)

		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid line")
	})
}

func TestFormat(t *testing.T) {
	t.Parallel()

	t.Run("formats hashes", func(t *testing.T) {
		t.Parallel()

		hashes := []string{"abc123", "def456"}
		got := string(Format(hashes))
		assert.Equal(t, "abc123\ndef456\n", got)
	})
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	hashes := []string{"abc123", "def456"}

	data := Format(hashes)
	parsed, err := Parse(data)

	require.NoError(t, err)
	assert.Equal(t, hashes, parsed)
}
