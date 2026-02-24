package sliceutil_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.inout.gg/conduit/internal/sliceutil"
)

func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("maps ints to strings", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []int{1, 2, 3, 4, 5}

		// Act
		result := sliceutil.Map(input, strconv.Itoa)

		// Assert
		expected := []string{"1", "2", "3", "4", "5"}
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var input []int

		// Act
		result := sliceutil.Map(input, func(i int) bool { return i%2 == 0 })

		// Assert
		assert.Empty(t, result)
	})
}

func TestKeyBy(t *testing.T) {
	t.Parallel()

	t.Run("keys strings by their length", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []string{"a", "ab", "abc", "abcd"}

		// Act
		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		// Assert
		expected := map[int]string{
			1: "a",
			2: "ab",
			3: "abc",
			4: "abcd",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("last value wins on overlapping keys", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []int{1, 2, 3, 4, 5}

		// Act
		result := sliceutil.KeyBy(input, func(i int) string {
			if i%2 == 0 {
				return "even"
			}

			return "odd"
		})

		// Assert
		expected := map[string]int{
			"odd":  5,
			"even": 4,
		}
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty map for empty input", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var input []string

		// Act
		result := sliceutil.KeyBy(input, func(s string) int { return len(s) })

		// Assert
		assert.Empty(t, result)
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()

	t.Run("filters even numbers", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []int{1, 2, 3, 4, 5}

		// Act
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		// Assert
		expected := []int{2, 4}
		assert.Equal(t, expected, result)
	})

	t.Run("returns all elements when all match", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []int{2, 4, 6}

		// Act
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		// Assert
		expected := []int{2, 4, 6}
		assert.Equal(t, expected, result)
	})

	t.Run("returns empty slice when none match", func(t *testing.T) {
		t.Parallel()

		// Arrange
		input := []int{1, 3, 5}

		// Act
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		// Assert
		assert.Empty(t, result)
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var input []int

		// Act
		result := sliceutil.Filter(input, func(i int) bool { return i%2 == 0 })

		// Assert
		assert.Empty(t, result)
	})
}
