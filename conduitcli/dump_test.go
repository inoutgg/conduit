package conduitcli

import (
	"bytes"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/testutil"
)

func TestDump(t *testing.T) {
	t.Parallel()

	const schema = `
CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE posts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users (id),
    title text NOT NULL
);
`

	t.Run("should return error, when database URL is invalid", func(t *testing.T) {
		t.Parallel()

		args := DumpArgs{DatabaseURL: "://invalid"}
		recorder := new(bytes.Buffer)

		err := Dump(t.Context(), recorder, bi, args)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse database URL")
	})

	t.Run("should write DDL to writer, when database has schema", func(t *testing.T) {
		t.Parallel()

		pool := poolFactory.Pool(t)
		testutil.Exec(t, pool, schema)

		args := DumpArgs{DatabaseURL: testutil.ConnString(pool)}
		recorder := new(bytes.Buffer)

		err := Dump(t.Context(), recorder, bi, args)
		require.NoError(t, err)

		output := recorder.String()

		require.NotEmpty(t, output)
		snaps.MatchSnapshot(t, output)
	})
}
