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
	"go.uber.org/mock/gomock"
)

func TestDirContextMethods(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tempDir := filepath.ToSlash(t.TempDir())
	store := NewMockStore(ctrl)
	store.EXPECT().RootURL().Return(url.URL{Scheme: "file"}).AnyTimes()
	store.EXPECT().RootTitle().Return("title").AnyTimes()
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	store.EXPECT().CreateDir(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	store.EXPECT().CreateFile(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	store.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	store.EXPECT().GetDirReader(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	dir := NewDirContext(store, tempDir, nil)

	assert.Equal(t, "title", dir.Store().RootTitle())
	_, _ = dir.Store().ReadDir(context.TODO(), "")
	_ = dir.Store().CreateDir(context.TODO(), "")
	_ = dir.Store().CreateFile(context.TODO(), "")
	_ = dir.Store().Delete(context.TODO(), "")
	_, _ = dir.Store().GetDirReader(context.TODO(), "")

	dr := NewMockDirReader(ctrl)
	dr.EXPECT().Close().Return(nil).AnyTimes()
	dr.EXPECT().Readdir().Return(nil, nil).AnyTimes()
	_ = dr.Close()
	_, _ = dr.Readdir()

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
	assert.Equal(t, tempDir, dir.Path())
	assert.Equal(t, "file", dir.Store().RootURL().Scheme)

	root := NewDirContext(nil, "/", nil)
	assert.Equal(t, "/", root.Name())
	assert.Nil(t, root.Store())
	assert.Equal(t, "/", root.Path())

	empty := NewDirContext(nil, "", nil)
	assert.Equal(t, "", empty.DirPath())
	assert.Equal(t, "", empty.Name())
	info, err = empty.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)
	assert.Equal(t, "", empty.Path())

	nonFileStore := NewMockStore(ctrl)
	nonFileStore.EXPECT().RootURL().Return(url.URL{Scheme: "ftp"}).AnyTimes()
	nonFileCtx := NewDirContext(nonFileStore, tempDir, nil)
	info, err = nonFileCtx.Info()
	assert.NoError(t, err)
	assert.Nil(t, info)
}

func TestDirContextChildrenReturnsCopy(t *testing.T) {
	t.Parallel()
	dirEntries := []os.DirEntry{NewDirEntry("a.txt", false)}
	dir := NewDirContext(nil, "", dirEntries)

	children := dir.Children()
	if assert.Len(t, children, 1) {
		assert.Equal(t, "a.txt", children[0].Name())
	}

	children[0] = NewDirEntry("b.txt", false)
	children = append(children, NewDirEntry("c.txt", false))
	assert.Len(t, children, 2)

	updatedChildren := dir.Children()
	if assert.Len(t, updatedChildren, 1) {
		assert.Equal(t, "a.txt", updatedChildren[0].Name())
	}
}

func TestDirContextChildrenNil(t *testing.T) {
	t.Parallel()
	dir := NewDirContext(nil, "", nil)

	assert.Nil(t, dir.Children())
}
