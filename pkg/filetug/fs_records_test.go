package filetug

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/datatug/filetug/pkg/files"
	"github.com/datatug/filetug/pkg/filetug/ftui"
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

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestNewFileRows(t *testing.T) {
	dir := &DirContext{Path: "/test"}
	fr := NewFileRows(dir)
	assert.NotNil(t, fr)
	assert.Equal(t, dir, fr.Dir)
}

func TestFileRows_SetFilter(t *testing.T) {
	fr := NewFileRows(&DirContext{})
	fr.AllEntries = []os.DirEntry{
		mockDirEntry{name: "file.txt", isDir: false},
		mockDirEntry{name: ".hidden", isDir: false},
		mockDirEntry{name: "dir", isDir: true},
	}
	fr.Infos = make([]os.FileInfo, len(fr.AllEntries))

	// Default filter: ShowHidden=false, ShowDirs=false
	fr.SetFilter(ftui.Filter{ShowHidden: false, ShowDirs: false})
	assert.Len(t, fr.VisibleEntries, 1)
	assert.Equal(t, "file.txt", fr.VisibleEntries[0].Name())

	// Show hidden
	fr.SetFilter(ftui.Filter{ShowHidden: true, ShowDirs: false})
	assert.Len(t, fr.VisibleEntries, 2)

	// Show dirs
	fr.SetFilter(ftui.Filter{ShowHidden: false, ShowDirs: true})
	assert.Len(t, fr.VisibleEntries, 2)
}

func TestFileRows_GetRowCount(t *testing.T) {
	store := mockStore{root: url.URL{Path: "/"}}
	fr := NewFileRows(&DirContext{Store: store, Path: "/home"})
	fr.VisibleEntries = []os.DirEntry{
		mockDirEntry{name: "f1", isDir: false},
	}
	fr.VisualInfos = make([]os.FileInfo, 1)
	// With parent row (..)
	assert.Equal(t, 2, fr.GetRowCount())

	// Hide parent row
	fr.Dir.Path = "/"
	assert.Equal(t, 1, fr.GetRowCount())
}

func TestFileRows_GetCell(t *testing.T) {
	store := mockStore{root: url.URL{Path: "/"}}
	fr := NewFileRows(&DirContext{Store: store, Path: "/home"})
	fr.VisibleEntries = []os.DirEntry{
		mockDirEntry{name: "file.go", isDir: false},
	}
	fr.VisualInfos = []os.FileInfo{
		files.NewFileInfo(files.NewDirEntry("file.go", false), files.Size(1024), files.ModTime(time.Now())),
	}

	// Row 0 is ".."
	cell := fr.GetCell(0, 0)
	assert.NotNil(t, cell)
	assert.Contains(t, cell.Text, "..")

	// Row 1 is "file.go"
	cell = fr.GetCell(1, 0)
	assert.NotNil(t, cell)
	assert.Contains(t, cell.Text, "file.go")

	// Column 1 is size
	cell = fr.GetCell(1, 1)
	assert.NotNil(t, cell)
	assert.Equal(t, "1KB", cell.Text)

	// Column 2 is modified time
	cell = fr.GetCell(1, 2)
	assert.NotNil(t, cell)
	assert.NotEmpty(t, cell.Text)
}
