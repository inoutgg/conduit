package sliceutil_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.inout.gg/conduit/internal/sliceutil"
)

func TestMap(t *testing.T) {
	t.Run("Map integers to strings", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := sliceutil.Map(input, strconv.Itoa)

		expected := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expected, result)
	})

	t.Run("Map strings to their lengths", func(t *testing.T) {
		input := []string{"apple", "banana", "cherry"}
		result := sliceutil.Map(input, func(s string) int { return len(s) })

		expected := []int{5, 6, 6}
		assert.Equal(t, expected, result)
	})

	t.Run("Map empty slice", func(t *testing.T) {
		var input []int
		result := sliceutil.Map(input, func(i int) bool { return i%2 == 0 })

		assert.Empty(t, result)
	})
}

func TestKeyBy(t *testing.T) {
	t.Run("Key strings by length", func(t *testing.T) {
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

	t.Run("Keys overlapping", func(t *testing.T) {
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

	t.Run("Key empty slice", func(t *testing.T) {
		var input []string
		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		assert.Empty(t, result)
	})
}
