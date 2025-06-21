package version_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/internal/version"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		expectedTime   time.Time
		name           string
		filename       string
		expectedName   string
		expectedExt    string
		expectedErrMsg string
	}{
		{
			name:         "Valid SQL migration",
			filename:     "20230601120000_create_user.sql",
			expectedTime: parseTime("20230601120000"),
			expectedName: "create_user",
			expectedExt:  "sql",
		},
		{
			name:         "Valid Go migration",
			filename:     "20230601130000_update_schema.go",
			expectedTime: parseTime("20230601130000"),
			expectedName: "update_schema",
			expectedExt:  "go",
		},
		{
			name:         "Valid migration with path",
			filename:     "/path/to/migrations/20230601140000_add_index.sql",
			expectedTime: parseTime("20230601140000"),
			expectedName: "add_index",
			expectedExt:  "sql",
		},
		{
			name:           "Invalid extension",
			filename:       "20230601150000_invalid_extension.txt",
			expectedErrMsg: `conduit: unknown migration file extension ".txt", expected: .sql or .go`,
		},
		{
			name:           "Malformed filename, no underscore",
			filename:       "20230601160000malformed.go",
			expectedErrMsg: `conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: 20230601160000malformed.go`,
		},
		{
			name:           "Malformed filename, only version",
			filename:       "20230601170000.go",
			expectedErrMsg: `conduit: malformed migration filename, expected: <version>_<name>.[go|sql], got: 20230601170000.go`,
		},
		{
			name:           "Non-numeric version",
			filename:       "abc_invalid_version.sql",
			expectedErrMsg: `conduit: invalid version format "abc", expected: YYYYMMDDHHMMSS`,
		},
		{
			name:           "Empty filename",
			filename:       "",
			expectedErrMsg: "conduit: filename cannot be empty",
		},
		{
			name:           "Invalid version format",
			filename:       "1234_invalid_version.sql",
			expectedErrMsg: "conduit: invalid version format \"1234\", expected: YYYYMMDDHHMMSS",
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
				assert.Equal(t, version.NewFromTime(tt.expectedTime), parsed.Version)
				assert.Equal(t, tt.expectedName, parsed.Name)
				assert.Equal(t, tt.expectedExt, parsed.Extension)
			}
		})
	}
}

// Helper function to parse time in the expected format.
func parseTime(timeStr string) time.Time {
	t, err := time.Parse("20060102150405", timeStr)
	if err != nil {
		panic(err)
	}

	return t
}
