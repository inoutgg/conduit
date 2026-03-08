// Package stopwatch provides an interface for measuring elapsed time,
// allowing injection of a fixed duration in tests.
package stopwatch

import "time"

// Stopwatch measures elapsed time. Call Start to begin timing; the returned
// function yields the elapsed duration when called.
type Stopwatch interface {
	Start() func() time.Duration
}

// Standard measures real wall-clock time.
type Standard struct{}

func (Standard) Start() func() time.Duration {
	start := time.Now()

	return func() time.Duration { return time.Since(start) }
}

// Stub always reports a fixed duration. Intended for use in tests.
type Stub struct{ D time.Duration }

func (s Stub) Start() func() time.Duration {
	return func() time.Duration { return s.D }
}
