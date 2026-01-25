package pgdiff

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// migrationsBuilder provides a builder for setting up
// an in-memory filesystem with a migrations directory.
type migrationsBuilder struct {
	fs            afero.Fs
	failOnFileErr error
	t             *testing.T
	dir           string
	failOnFile    string
}

func newMigrationsBuilder(t *testing.T) *migrationsBuilder {
	t.Helper()

	fs := afero.NewMemMapFs()
	dir := "/migrations"
	require.NoError(t, fs.MkdirAll(dir, 0o755))

	return &migrationsBuilder{t: t, fs: fs, dir: dir}
}

func (b *migrationsBuilder) WithFile(name, content string) *migrationsBuilder {
	b.t.Helper()
	require.NoError(b.t, afero.WriteFile(b.fs, filepath.Join(b.dir, name), []byte(content), 0o644))

	return b
}

func (b *migrationsBuilder) WithSubdir(name string) *migrationsBuilder {
	b.t.Helper()
	require.NoError(b.t, b.fs.MkdirAll(filepath.Join(b.dir, name), 0o755))

	return b
}

func (b *migrationsBuilder) WithReadError(file string, err error) *migrationsBuilder {
	b.t.Helper()
	b.failOnFile = filepath.Join(b.dir, file)
	b.failOnFileErr = err

	return b
}

func (b *migrationsBuilder) Build() (afero.Fs, string) {
	b.t.Helper()

	fs := b.fs
	if b.failOnFile != "" {
		fs = &readErrorFs{
			Fs:   b.fs,
			file: b.failOnFile,
			err:  b.failOnFileErr,
		}
	}

	return fs, b.dir
}

// readErrorFs wraps an afero.Fs and returns an error when reading a specific file.
type readErrorFs struct {
	afero.Fs

	err  error
	file string
}

func (f *readErrorFs) Open(name string) (afero.File, error) {
	if name == f.file {
		return nil, f.err
	}

	//nolint:wrapcheck
	return f.Fs.Open(name)
}
