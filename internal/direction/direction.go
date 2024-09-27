package direction

import "errors"

// Direction denotes whether SQL migration should be rolled up, or rolled back.
type Direction string

const (
	DirectionUp   Direction = "up"   // rollup
	DirectionDown           = "down" // rollback
)

var UnknownDirectionErr = errors.New("conduit: unknown direction")
