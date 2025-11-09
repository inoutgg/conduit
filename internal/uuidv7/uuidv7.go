package uuidv7

import (
	"github.com/gofrs/uuid/v5"
	"go.inout.gg/foundations/must"
)

// Must returns a new random UUID. It panics if there is an error.
func Must() uuid.UUID {
	return must.Must(uuid.NewV7())
}
