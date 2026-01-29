package files

import (
	"context"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	root url.URL
}

func (m mockStore) RootTitle() string { return "Mock" }
func (m mockStore) RootURL() url.URL  { return m.root }
func (m mockStore) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	_, _ = ctx, name
	return nil, nil
}
func (m mockStore) CreateDir(ctx context.Context, path string) error {
	_, _ = ctx, path
	return nil
}
func (m mockStore) CreateFile(ctx context.Context, path string) error {
	_, _ = ctx, path
	return nil
}
func (m mockStore) Delete(ctx context.Context, path string) error {
	_, _ = ctx, path
	return nil
}

func TestDirContextMethods(t *testing.T) {
	tempDir := filepath.ToSlash(t.TempDir())
	dir := NewDirContext(mockStore{root: url.URL{Scheme: "file"}}, tempDir, nil)

	beforeSet := time.Now()
	dir.SetChildren([]os.DirEntry{NewDirEntry("a.txt", false)})
	afterSet := time.Now()
	assert.Len(t, dir.Children(), 1)
	timestamp := dir.Timestamp()
	assert.False(t, timestamp.IsZero())
	onOrAfter := timestamp.After(beforeSet) || timestamp.Equal(beforeSet)
	assert.True(t, onOrAfter)
	onOrBefore := timestamp.Before(afterSet) || timestamp.Equal(afterSet)
	assert.True(t, onOrBefore)

	entries := dir.Entries()
	if assert.Len(t, entries, 1) {
		assert.Equal(t, "a.txt", entries[0].Name())
		assert.Equal(t, tempDir, entries[0].DirPath())
	}

	assert.Equal(t, path.Dir(tempDir), dir.DirPath())
	assert.Equal(t, tempDir, dir.FullName())
	assert.Equal(t, tempDir, dir.String())
	assert.Equal(t, path.Base(tempDir), dir.Name())
	assert.True(t, dir.IsDir())
	assert.Equal(t, os.ModeDir, dir.Type())
	info, err := dir.Info()
	assert.NoError(t, err)
	assert.NotNil(t, info)

	root := NewDirContext(nil, "/", nil)
	assert.Equal(t, "/", root.Name())

	empty := NewDirContext(nil, "", nil)
	assert.Equal(t, "", empty.DirPath())
	assert.Equal(t, "", empty.Name())
	info, err = empty.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)

	nonFileStore := mockStore{root: url.URL{Scheme: "ftp"}}
	nonFileCtx := NewDirContext(nonFileStore, tempDir, nil)
	info, err = nonFileCtx.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)
}
