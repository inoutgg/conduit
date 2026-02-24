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
