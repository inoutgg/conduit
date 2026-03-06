// Package conduiterror provides structured error display for the conduit CLI.
package conduiterror

import (
	"errors"
	"fmt"
	"io"

	"go.inout.gg/conduit"
)

// Display writes a formatted error to w.
//
// For known sentinel errors it appends contextual hints to help the user
// resolve the issue. The output format is:
//
//	Error: <message>
//
//	Hint: <hint>
func Display(w io.Writer, err error) {
	var hint string

	switch {
	case errors.Is(err, conduit.ErrSchemaDrift):
		hint = "if this is expected, re-run with --skip-schema-drift-check"
	case errors.Is(err, conduit.ErrHazardDetected):
		hint = "use --allow-hazards <TYPE> to allow specific hazard types"
	}

	fmt.Fprintf(w, "Error: %s\n", err)

	if hint != "" {
		fmt.Fprintf(w, "\nHint: %s\n", hint)
	}
}
