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
		hint = "the database schema was modified outside of migrations (manual DDL).\n" +
			"Run 'conduit diff' to generate a migration that captures the changes.\n" +
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
