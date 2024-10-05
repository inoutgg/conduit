package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.inout.gg/conduit/internal/version"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedVer    int64
		expectedName   string
		expectedExt    string
		expectedErrMsg string
	}{
		{
			name:         "Valid SQL migration",
			filename:     "1257894000000_create_user.sql",
			expectedVer:  1257894000000,
			expectedName: "create_user",
			expectedExt:  "sql",
		},
		{
			name:         "Valid Go migration",
			filename:     "1257894454320_update_schema.go",
			expectedVer:  1257894454320,
			expectedName: "update_schema",
			expectedExt:  "go",
		},
		{
			name:         "Valid migration with path",
			filename:     "/path/to/migrations/1257894900000_add_index.sql",
			expectedVer:  1257894900000,
			expectedName: "add_index",
			expectedExt:  "sql",
		},
		{
			name:           "Invalid extension",
			filename:       "1257895000000_invalid_extension.txt",
			expectedErrMsg: `conduit: unknown migration file extension ".txt", expected: .sql or .go`,
		},
		{
			name:           "Malformed filename, no underscore",
			filename:       "1257895100000malformed.go",
			expectedErrMsg: `conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: 1257895100000malformed.go`,
		},
		{
			name:           "Malformed filename, only version",
			filename:       "1257895200000.go",
			expectedErrMsg: `conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: 1257895200000.go`,
		},
		{
			name:           "Non-numeric version",
			filename:       "abc_invalid_version.sql",
			expectedErrMsg: `conduit: unable to parse version "abc":`,
		},
		{
			name:           "Empty filename",
			filename:       "",
			expectedErrMsg: "conduit: filename cannot be empty",
		},
		{
			name:           "Invalid version (negative)",
			filename:       "-1_invalid_version.sql",
			expectedErrMsg: "conduit: invalid version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := version.ParseMigrationFilename(tt.filename)

			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVer, parsed.Version)
				assert.Equal(t, tt.expectedName, parsed.Name)
				assert.Equal(t, tt.expectedExt, parsed.Extension)
			}
		})
	}
}
