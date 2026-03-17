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
		hint = "one or more migrations have been modified since the lockfile was generated.\n" +
			"Run 'conduit rehash' to recompute conduit.lock from the current migrations.\n" +
			"Run 'conduit validate' to identify the exact migration that diverged.\n" +
			"To skip this check: --skip-schema-drift-check"
	case errors.Is(err, conduit.ErrHazardDetected):
		hint = "these operations can cause table locks, downtime, or irreversible data loss in production.\n" +
			"Review each hazard above before proceeding.\n" +
			"To explicitly allow specific types: --allow-hazards <TYPE>"
	}

	fmt.Fprintf(w, "Error: %s\n", err)

	if hint != "" {
		fmt.Fprintf(w, "\nHint: %s\n", hint)
	}
}
