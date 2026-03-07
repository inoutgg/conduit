package conduitversion_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.inout.gg/conduit/pkg/conduitversion"
)

func TestVersion_Compare(t *testing.T) {
	t.Parallel()

	t.Run("should return negative, when earlier conduitversion compared to later", func(t *testing.T) {
		t.Parallel()

		// Arrange
		earlier := conduitversion.NewFromTime(parseTime("20230601120000"))
		later := conduitversion.NewFromTime(parseTime("20230601130000"))

		// Act
		result1 := earlier.Compare(later)
		result2 := later.Compare(earlier)

		// Assert
		assert.Equal(t, -1, result1)
		assert.Equal(t, 1, result2)
	})

	t.Run("should return zero, when conduitversions are equal", func(t *testing.T) {
		t.Parallel()

		// Arrange
		v1 := conduitversion.NewFromTime(parseTime("20230601120000"))
		v2 := conduitversion.NewFromTime(parseTime("20230601120000"))

		// Act
		result := v1.Compare(v2)

		// Assert
		assert.Equal(t, 0, result)
	})
}

func TestParseMigrationFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedTime   time.Time
		name           string
		filename       string
		expectedName   string
		expectedDir    conduitversion.MigrationDirection
		expectedErrMsg string
	}{
		{
			name:         "Valid up migration",
			filename:     "20230601120000_create_user.up.sql",
			expectedTime: parseTime("20230601120000"),
			expectedName: "create_user",
			expectedDir:  conduitversion.MigrationDirectionUp,
		},
		{
			name:         "Valid down migration",
			filename:     "20230601120000_create_user.down.sql",
			expectedTime: parseTime("20230601120000"),
			expectedName: "create_user",
			expectedDir:  conduitversion.MigrationDirectionDown,
		},
		{
			name:         "Valid up migration with path",
			filename:     "/path/to/migrations/20230601140000_add_index.up.sql",
			expectedTime: parseTime("20230601140000"),
			expectedName: "add_index",
			expectedDir:  conduitversion.MigrationDirectionUp,
		},
		{
			name:           "Invalid extension",
			filename:       "20230601150000_invalid_extension.txt",
			expectedErrMsg: `unknown migration file extension ".txt", expected: .sql`,
		},
		{
			name:           "Malformed filename, no underscore",
			filename:       "20230601160000malformed.up.sql",
			expectedErrMsg: `malformed migration filename, expected: <version>_<name>.sql, got: 20230601160000malformed.up.sql`,
		},
		{
			name:           "SQL without direction suffix",
			filename:       "20230601120000_create_user.sql",
			expectedErrMsg: `must have .up.sql or .down.sql suffix`,
		},
		{
			name:           "Non-numeric conduitversion",
			filename:       "abc_invalid_conduitversion.up.sql",
			expectedErrMsg: `invalid version format "abc", expected: YYYYMMDDHHMMSS`,
		},
		{
			name:           "Empty filename",
			filename:       "",
			expectedErrMsg: "filename cannot be empty",
		},
		{
			name:           "Invalid conduitversion format",
			filename:       "1234_invalid_conduitversion.up.sql",
			expectedErrMsg: "invalid version format \"1234\", expected: YYYYMMDDHHMMSS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			parsed, err := conduitversion.ParseMigrationFilename(tt.filename)

			// Assert
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, conduitversion.NewFromTime(tt.expectedTime), parsed.Version)
				assert.Equal(t, tt.expectedName, parsed.Name)
				assert.Equal(t, tt.expectedDir, parsed.Direction)
			}
		})
	}
}

func TestParsedMigrationFilename_Filename(t *testing.T) {
	t.Parallel()

	t.Run("should return original filename, when parsed from up migration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		original := "20230601120000_create_user.up.sql"
		parsed, err := conduitversion.ParseMigrationFilename(original)
		require.NoError(t, err)

		// Act
		result := parsed.Filename()

		// Assert
		assert.Equal(t, original, result)
	})

	t.Run("should return original filename, when parsed from down migration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		original := "20230601120000_create_user.down.sql"
		parsed, err := conduitversion.ParseMigrationFilename(original)
		require.NoError(t, err)

		// Act
		result := parsed.Filename()

		// Assert
		assert.Equal(t, original, result)
	})

	t.Run("should strip path, when filename includes directory prefix", func(t *testing.T) {
		t.Parallel()

		// Arrange
		parsed, err := conduitversion.ParseMigrationFilename(
			"/path/to/migrations/20230601140000_add_index.up.sql")
		require.NoError(t, err)

		// Act
		result := parsed.Filename()

		// Assert
		assert.Equal(t, "20230601140000_add_index.up.sql", result)
	})
}

// parseTime is helper function to parse time in the expected format.
func parseTime(timeStr string) time.Time {
	t, err := time.Parse("20060102150405", timeStr)
	if err != nil {
		panic(err)
	}

	return t
}
