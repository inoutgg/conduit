package direction

import (
	"errors"
	"fmt"
)

// Direction represents the SQL migration direction: up (rollup) or down (rollback).
type Direction string

const (
	DirectionUp   Direction = "up"   // rolling up
	DirectionDown Direction = "down" // rolling back
)

var ErrUnknownDirection = errors.New("unknown direction: expected 'up' or 'down'")

// FromString converts a string to a Direction. It returns UnknownDirectionErr for invalid inputs.
func FromString(s string) (Direction, error) {
	switch s {
	case string(DirectionUp):
		return DirectionUp, nil
	case string(DirectionDown):
		return DirectionDown, nil
	}

	return "", fmt.Errorf("%w: got %q", ErrUnknownDirection, s)
}
