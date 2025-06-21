package sliceutil_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.inout.gg/conduit/internal/sliceutil"
)

func TestMap(t *testing.T) {
	t.Run("Map should work", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := sliceutil.Map(input, strconv.Itoa)

		expected := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expected, result)
	})

	t.Run("Map empty slice", func(t *testing.T) {
		var input []int
		result := sliceutil.Map(input, func(i int) bool { return i%2 == 0 })

		assert.Empty(t, result)
	})
}

func TestKeyBy(t *testing.T) {
	t.Run("KeyBy should work", func(t *testing.T) {
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
		var input []string
		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		assert.Empty(t, result)
	})
}

func TestSome(t *testing.T) {
	t.Run("Some should work", func(t *testing.T) {
		input := []int{2, 4}

		assert.True(t, sliceutil.Some(input, func(i int) bool { return i%2 == 0 }))
		assert.False(t, sliceutil.Some(input, func(i int) bool { return i%3 == 0 }))
	})

	t.Run("Some empty slice", func(t *testing.T) {
		var input []int
		result := sliceutil.Some(input, func(i int) bool { return i%2 == 0 })

		assert.False(t, result)
	})
}
