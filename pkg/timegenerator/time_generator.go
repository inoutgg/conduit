// Package timegenerator provides an interface for generating the current time,
// allowing injection of a fixed clock in tests.
package timegenerator

import "time"

// Generator is an interface for providing the current time.
type Generator interface {
	Now() time.Time
}

// Standard is a Generator that returns the current time.
type Standard struct{}

func (Standard) Now() time.Time { return time.Now() }

// Stub is a Generator that always returns a fixed time.
// This is intended for use in tests.
type Stub struct{ T time.Time }

func (s Stub) Now() time.Time { return s.T }
