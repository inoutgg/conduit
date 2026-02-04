// Package sliceutil provides utility functions for working with slices.
package sliceutil

// Map applies function f to each element of slice s and
// returns the results as a new slice.
func Map[S ~[]E, E, V any](s S, f func(E) V) []V {
	values := make([]V, len(s))
	for i, e := range s {
		values[i] = f(e)
	}

	return values
}

// KeyBy creates a map from slice s, using function f to generate keys for each element.
// The resulting map consists of key-value pairs where the key is f(e) and the value is e.
func KeyBy[S ~[]E, E any, V comparable](s S, f func(E) V) map[V]E {
	m := make(map[V]E, len(s))

	for _, e := range s {
		k := f(e)
		m[k] = e
	}

	return m
}

// Filter returns a new slice containing only elements for which f returns true.
func Filter[S ~[]E, E any](s S, f func(E) bool) S {
	result := make(S, 0, len(s))
	for _, e := range s {
		if f(e) {
			result = append(result, e)
		}
	}

	return result
}
