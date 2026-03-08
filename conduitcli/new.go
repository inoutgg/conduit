package conduitcli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"

	"go.inout.gg/conduit/pkg/conduitversion"
	"go.inout.gg/conduit/pkg/timegenerator"
)

// NewArgs configures a [New] operation.
type NewArgs struct {
	MigrationsDir string
	Name          string
}

// NewResult holds the paths created by [New].
type NewResult struct {
	UpFile   string
	DownFile string
}

// New creates a pair of empty up and down migration files in the migrations
// directory.
func New(fs afero.Fs, timeGen timegenerator.Generator, args NewArgs) (*NewResult, error) {
	if !exists(fs, args.MigrationsDir) {
		return nil, fmt.Errorf("%w: directory %q does not exist",
			ErrMigrationsNotFound, args.MigrationsDir)
	}

	migrationsFs := afero.NewBasePathFs(fs, args.MigrationsDir)
	v := conduitversion.NewFromTime(timeGen.Now())

	upFilename := conduitversion.MigrationFilename(v, args.Name, conduitversion.MigrationDirectionUp)
	downFilename := conduitversion.MigrationFilename(v, args.Name, conduitversion.MigrationDirectionDown)

	if err := afero.WriteFile(migrationsFs, upFilename, nil, 0o644); err != nil {
		return nil, fmt.Errorf("failed to create up migration: %w", err)
	}

	if err := afero.WriteFile(migrationsFs, downFilename, nil, 0o644); err != nil {
		return nil, fmt.Errorf("failed to create down migration: %w", err)
	}

	return &NewResult{
		UpFile:   filepath.Join(args.MigrationsDir, upFilename),
		DownFile: filepath.Join(args.MigrationsDir, downFilename),
	}, nil
}
