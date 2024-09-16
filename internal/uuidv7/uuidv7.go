// uuidv7 is a wrapper around google's uuid package.
package uuidv7

import (
	"github.com/google/uuid"
	"go.inout.gg/foundations/must"
)

// Must returns a new random UUID. It panics if there is an error.
func Must() uuid.UUID {
	return must.Must(uuid.NewV7())
}

// FromString parses a UUID from a string.
func FromString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
