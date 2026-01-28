package viewers

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
)

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string { return m.name }
func (m mockDirEntry) IsDir() bool  { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode {
	if m.isDir {
		return fs.ModeDir
	}
	return 0
}
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestTextPreviewerReadFileError(t *testing.T) {
	previewer := NewTextPreviewer()
	tmpDir, _ := os.MkdirTemp("", "testdir")
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	entry := files.NewEntryWithDirPath(
		mockDirEntry{name: filepath.Base(tmpDir), isDir: true},
		filepath.Dir(tmpDir),
	)
	_, err := previewer.readFile(entry, 0)
	assert.Error(t, err)
}

func TestTextPreviewerReadFile(t *testing.T) {
	previewer := NewTextPreviewer()
	tmpFile, _ := os.CreateTemp("", "test*.txt")
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	content := "0123456789"
	err := os.WriteFile(tmpFile.Name(), []byte(content), 0644)
	assert.NoError(t, err)

	entry := files.NewEntryWithDirPath(
		mockDirEntry{name: filepath.Base(tmpFile.Name())},
		filepath.Dir(tmpFile.Name()),
	)

	data, err := previewer.readFile(entry, 0)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	data, err = previewer.readFile(entry, 5)
	assert.NoError(t, err)
	assert.Equal(t, "01234", string(data))

	data, err = previewer.readFile(entry, -5)
	assert.NoError(t, err)
	assert.Equal(t, "56789", string(data))

	data, err = previewer.readFile(entry, -20)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
}
