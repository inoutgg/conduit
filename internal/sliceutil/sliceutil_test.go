package sliceutil_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.inout.gg/conduit/internal/sliceutil"
)

func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("Map should work", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		result := sliceutil.Map(input, strconv.Itoa)

		expected := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expected, result)
	})

	t.Run("Map empty slice", func(t *testing.T) {
		t.Parallel()

		var input []int

		result := sliceutil.Map(input, func(i int) bool { return i%2 == 0 })

		assert.Empty(t, result)
	})
}

func TestKeyBy(t *testing.T) {
	t.Parallel()

	t.Run("KeyBy should work", func(t *testing.T) {
		t.Parallel()

		input := []string{"a", "ab", "abc", "abcd"}
		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		expected := map[int]string{
			1: "a",
			2: "ab",
			3: "abc",
			4: "abcd",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("KeyBy overlapping keys", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		result := sliceutil.KeyBy(input, func(i int) string {
			if i%2 == 0 {
				return "even"
			}

			return "odd"
		})

		expected := map[string]int{
			"odd":  5,
			"even": 4,
		}
		assert.Equal(t, expected, result)
	})

	t.Run("KeyBy empty slice", func(t *testing.T) {
		t.Parallel()

		var input []string

		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		assert.Empty(t, result)
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()

	t.Run("Filter should work", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 2, 3, 4, 5}
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		expected := []int{2, 4}
		assert.Equal(t, expected, result)
	})

	t.Run("Filter all match", func(t *testing.T) {
		t.Parallel()

		input := []int{2, 4, 6}
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		expected := []int{2, 4, 6}
		assert.Equal(t, expected, result)
	})

	t.Run("Filter none match", func(t *testing.T) {
		t.Parallel()

		input := []int{1, 3, 5}
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		assert.Empty(t, result)
	})

	t.Run("Filter empty slice", func(t *testing.T) {
		t.Parallel()

		var input []int

		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		assert.Empty(t, result)
	})
}
