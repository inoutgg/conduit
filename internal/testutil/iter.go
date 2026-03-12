package testutil

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/require"
)

// CollectSeq2 collects all values from an [iter.Seq2] iterator.
// It fails the test if any iteration yields a non-nil error.
func CollectSeq2[V any](tb testing.TB, seq iter.Seq2[V, error]) []V {
	tb.Helper()

	results := make([]V, 0)

	for v, err := range seq {
		require.NoError(tb, err)

		results = append(results, v)
	}

	return results
}

// CollectSeq2Error iterates an [iter.Seq2] until an error is encountered
// and returns it. Returns nil if the iterator completes without error.
func CollectSeq2Error[V any](tb testing.TB, seq iter.Seq2[V, error]) error {
	tb.Helper()

	for _, err := range seq {
		if err != nil {
			return err
		}
	}

	return nil
}
