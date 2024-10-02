package direction

import (
	"errors"
)

// Direction denotes whether SQL migration should be rolled up, or rolled back.
type Direction string

const (
	DirectionUp   Direction = "up"   // rollup
	DirectionDown           = "down" // rollback
)

var UnknownDirectionErr = errors.New("conduit: unknown direction")

// FromString converts a string to a direction if possible, otherwise
// it returns an error
func FromString(s string) (Direction, error) {
	switch s {
	case string(DirectionUp):
		return DirectionUp, nil
	case string(DirectionDown):
		return DirectionDown, nil
	}

	return "", UnknownDirectionErr
}
