package sliceutil

// Map passes each element of the slice s to the function f and returns
// result as a slice.
func Map[S ~[]E, E, V any](s S, f func(E) V) []V {
	values := make([]V, len(s))
	for i, e := range s {
		values[i] = f(e)
	}

	return values
}

// KeyBy passes each element of the slices s to the function f and
// returns a map consisting of (f(e), e) pairs, where e is a single entry in s.
func KeyBy[S ~[]E, E any, V comparable](s S, f func(E) V) map[V]E {
	m := make(map[V]E, len(s))
	for _, e := range s {
		k := f(e)
		m[k] = e
	}

	return m
}
