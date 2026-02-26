package testutil

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type MigrationsDirBuilder struct {
	fs            afero.Fs
	failOnFileErr error
	t             *testing.T
	baseDir       string
	dir           string
	failOnFile    string
}

func NewMigrationsDirBuilder(t *testing.T) *MigrationsDirBuilder {
	t.Helper()

	fs := afero.NewMemMapFs()
	baseDir := "/testdir"
	dir := filepath.Join(baseDir, "migrations")
	require.NoError(t, fs.MkdirAll(baseDir, 0o755))
	require.NoError(t, fs.MkdirAll(dir, 0o755))

	//nolint:exhaustruct
	return &MigrationsDirBuilder{t: t, fs: fs, baseDir: baseDir, dir: dir}
}

func (b *MigrationsDirBuilder) WithFile(name, content string) *MigrationsDirBuilder {
	b.t.Helper()
	require.NoError(b.t, afero.WriteFile(b.fs, filepath.Join(b.dir, name), []byte(content), 0o644))

	return b
}

func (b *MigrationsDirBuilder) WithBaseFile(name, content string) *MigrationsDirBuilder {
	b.t.Helper()
	require.NoError(b.t, afero.WriteFile(b.fs, filepath.Join(b.baseDir, name), []byte(content), 0o644))

	return b
}

func (b *MigrationsDirBuilder) WithSubdir(name string) *MigrationsDirBuilder {
	b.t.Helper()
	require.NoError(b.t, b.fs.MkdirAll(filepath.Join(b.dir, name), 0o755))

	return b
}

func (b *MigrationsDirBuilder) WithReadError(file string, err error) *MigrationsDirBuilder {
	b.t.Helper()
	b.failOnFile = filepath.Join(b.dir, file)
	b.failOnFileErr = err

	return b
}

func (b *MigrationsDirBuilder) Build() (afero.Fs, string, string) {
	b.t.Helper()

	fs := b.fs
	if b.failOnFile != "" {
		fs = &readErrorFs{
			Fs:   b.fs,
			file: b.failOnFile,
			err:  b.failOnFileErr,
		}
	}

	return fs, b.baseDir, b.dir
}

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
