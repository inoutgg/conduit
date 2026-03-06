package conduiterror_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"go.inout.gg/conduit"
	"go.inout.gg/conduit/cmd/internal/conduiterror"
)

func TestDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		name string
	}{
		{
			name: "plain error",
			err:  errors.New("something went wrong"),
		},
		{
			name: "schema drift",
			err:  fmt.Errorf("%w: expected hash abc, got xyz", conduit.ErrSchemaDrift),
		},
		{
			name: "hazard detected",
			err: fmt.Errorf(
				"%w: migration 20250101_foo contains hazards:\n  - DELETES_DATA: drops column",
				conduit.ErrHazardDetected,
			),
		},
		{
			name: "wrapped schema drift",
			err: fmt.Errorf(
				"apply failed: %w",
				fmt.Errorf("%w: expected hash abc, got xyz", conduit.ErrSchemaDrift),
			),
		},
		{
			name: "wrapped hazard",
			err: fmt.Errorf(
				"apply failed: %w",
				fmt.Errorf("%w: migration foo contains hazards", conduit.ErrHazardDetected),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			conduiterror.Display(&buf, tt.err)

			snaps.MatchSnapshot(t, buf.String())
		})
	}
}
