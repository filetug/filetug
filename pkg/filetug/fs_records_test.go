package filetug

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
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

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string      { return m.name }
func (m mockDirEntry) IsDir() bool       { return m.isDir }
func (m mockDirEntry) Type() os.FileMode { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) {
	if m.name == "error.txt" {
		return nil, assert.AnError
	}
	return nil, nil
}

func TestNewFileRows(t *testing.T) {
	dir := &DirContext{Path: "/test"}
	fr := NewFileRows(dir)
	assert.NotNil(t, fr)
	assert.Equal(t, dir, fr.Dir)
}

func TestFileRows_SetFilter(t *testing.T) {
	fr := NewFileRows(&DirContext{})
	fr.AllEntries = []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "file.txt", isDir: false}},
		{DirEntry: mockDirEntry{name: ".hidden", isDir: false}},
		{DirEntry: mockDirEntry{name: "dir", isDir: true}},
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
	fr.VisibleEntries = []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "f1", isDir: false}},
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
	fr.VisibleEntries = []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "file.go", isDir: false}, Dir: "/home"},
	}
	entry := files.NewDirEntry("file.go", false)
	modTime := files.ModTime(time.Now())
	info := files.NewFileInfo(entry, files.Size(1024), modTime)
	fr.VisualInfos = []os.FileInfo{info}

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

func TestFileRows_getTopRowNameParentReference(t *testing.T) {
	store := mockStore{root: url.URL{Path: "/"}}
	dir := &DirContext{Store: store, Path: "/home/user"}
	fr := NewFileRows(dir)

	cell := fr.getTopRowName()
	ref := cell.GetReference()

	entryRef, ok := ref.(*files.EntryWithDirPath)
	if !assert.True(t, ok) {
		return
	}
	if !assert.NotNil(t, entryRef) {
		return
	}
	dirPath := entryRef.Dir
	name := entryRef.Name()
	assert.Equal(t, "/", dirPath)
	assert.Equal(t, "home", name)
}

func TestFileRows_SetGitStatusText(t *testing.T) {
	store := mockStore{root: url.URL{Path: "/"}}
	fr := NewFileRows(&DirContext{Store: store, Path: "/home"})
	fr.VisibleEntries = []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "file.go", isDir: false}, Dir: "/home"},
	}
	fr.VisualInfos = []os.FileInfo{
		files.NewFileInfo(files.NewDirEntry("file.go", false), files.Size(1024), files.ModTime(time.Now())),
	}

	fullPath := filepath.Join("/home", "file.go")
	statusText := "[gray]┆[-]main"
	updated := fr.SetGitStatusText(fullPath, statusText)
	assert.True(t, updated)

	updated = fr.SetGitStatusText(fullPath, statusText)
	assert.False(t, updated)

	cell := fr.GetCell(1, 0)
	assert.NotNil(t, cell)
	assert.Contains(t, cell.Text, "file.go")
	assert.Contains(t, cell.Text, "┆")

	updated = fr.SetGitStatusText(fullPath, "")
	assert.True(t, updated)

	cell = fr.GetCell(1, 0)
	assert.NotNil(t, cell)
	assert.Contains(t, cell.Text, "file.go")
	assert.NotContains(t, cell.Text, "┆")
}

func TestFileRows_Extra(t *testing.T) {
	store := mockStore{root: url.URL{Path: "/"}}
	fr := NewFileRows(&DirContext{Store: store, Path: "/"})
	fr.VisibleEntries = []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "dir1", isDir: true}},
	}
	fr.VisualInfos = make([]os.FileInfo, 1)

	t.Run("GetColumnCount", func(t *testing.T) {
		assert.Equal(t, 3, fr.GetColumnCount())
	})

	t.Run("GetCell_Error", func(t *testing.T) {
		fr.Err = assert.AnError
		cell := fr.GetCell(0, 0)
		assert.NotNil(t, cell)
		assert.Contains(t, cell.Text, "📁")
		fr.Err = nil
	})

	t.Run("GetCell_Empty", func(t *testing.T) {
		fr.VisibleEntries = nil
		fr.VisualInfos = nil
		fr.Dir.Path = "/home"    // Ensure HideParent() is false, so row 0 is parent row
		cell := fr.GetCell(1, 0) // Row 0 is parent, Row 1 is "No entries"
		assert.NotNil(t, cell)
		assert.Contains(t, cell.Text, "No entries")
	})

	t.Run("getTopRow", func(t *testing.T) {
		fr.Dir.Path = "/home"
		cell := fr.getTopRow(0)
		assert.Equal(t, "..", cell.Text)

		fr.Dir.Path = "/"
		cell = fr.getTopRow(0)
		assert.Equal(t, ".", cell.Text)

		fr.Dir.Path = "~"
		cell = fr.getTopRow(0)
		assert.Equal(t, "..", cell.Text)

		cell = fr.getTopRow(1)
		assert.Equal(t, "", cell.Text)

		cell = fr.getTopRow(2)
		assert.Equal(t, "", cell.Text)

		cell = fr.getTopRow(3)
		assert.Nil(t, cell)
	})

	t.Run("GetCell_Coverage_Gap", func(t *testing.T) {
		fr.Dir.Path = "/"
		fr.hideParent = true // So HideParent() returns true

		// i < 0
		assert.Nil(t, fr.GetCell(-1, 0))

		// i >= len(r.VisibleEntries)
		fr.VisibleEntries = []files.EntryWithDirPath{{DirEntry: mockDirEntry{name: "f1"}}}
		assert.Nil(t, fr.GetCell(2, 0))

		// Err != nil and col != nameColIndex
		fr.Err = assert.AnError
		assert.Nil(t, fr.GetCell(0, 1))
		fr.Err = nil

		// len(VisibleEntries) == 0 and col != nameColIndex
		fr.VisibleEntries = nil
		assert.Nil(t, fr.GetCell(0, 1))

		// dirEntry.IsDir() true for column 0
		fr.VisibleEntries = []files.EntryWithDirPath{{DirEntry: mockDirEntry{name: "my_dir", isDir: true}}}
		fr.VisualInfos = make([]os.FileInfo, 1)
		cell := fr.GetCell(0, 0)
		assert.Contains(t, cell.Text, "📁")

		// fi == nil or reflect.ValueOf(fi).IsNil() -> triggers Info()
		fr.VisualInfos = make([]os.FileInfo, 1)
		fr.Infos = make([]os.FileInfo, 1)
		cell = fr.GetCell(0, 1) // Column 1 triggers fi check
		assert.NotNil(t, cell)

		// dirEntry.Info() error
		fr.VisibleEntries = []files.EntryWithDirPath{{DirEntry: mockDirEntry{name: "error.txt"}}}
		fr.VisualInfos = make([]os.FileInfo, 1)
		cell = fr.GetCell(0, 1)
		assert.NotNil(t, cell)

		// fi.ModTime() in the future
		fr.VisibleEntries = []files.EntryWithDirPath{{DirEntry: mockDirEntry{name: "future.txt"}}}
		futureTime := time.Now().Add(48 * time.Hour)
		fr.VisualInfos = []os.FileInfo{
			files.NewFileInfo(files.NewDirEntry("future.txt", false), files.ModTime(futureTime)),
		}
		cell = fr.GetCell(0, 2)
		assert.Equal(t, futureTime.Format("15:04:05"), cell.Text)

		// col out of range
		assert.Nil(t, fr.GetCell(0, 10))
	})
}

func TestFileRows_isSymlinkToDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior depends on Windows permissions")
	}
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target-dir")
	err := os.Mkdir(targetDir, 0o755)
	assert.NoError(t, err)

	targetFile := filepath.Join(tmpDir, "target.txt")
	file, err := os.Create(targetFile)
	assert.NoError(t, err)
	if err == nil {
		closeErr := file.Close()
		assert.NoError(t, closeErr)
	}

	linkDir := filepath.Join(tmpDir, "link-dir")
	if err := os.Symlink(targetDir, linkDir); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	linkFile := filepath.Join(tmpDir, "link-file")
	err = os.Symlink(targetFile, linkFile)
	assert.NoError(t, err)

	brokenLink := filepath.Join(tmpDir, "broken-link")
	missingPath := filepath.Join(tmpDir, "missing")
	err = os.Symlink(missingPath, brokenLink)
	assert.NoError(t, err)

	entries, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)

	findEntry := func(name string) files.EntryWithDirPath {
		for _, entry := range entries {
			if entry.Name() == name {
				return files.EntryWithDirPath{DirEntry: entry, Dir: tmpDir}
			}
		}
		t.Fatalf("missing entry %s", name)
		return files.EntryWithDirPath{}
	}

	fileStore := mockStore{root: url.URL{Scheme: "file"}}
	rows := NewFileRows(&DirContext{Store: fileStore, Path: tmpDir})

	t.Run("dir symlink", func(t *testing.T) {
		entry := findEntry("link-dir")
		assert.True(t, rows.isSymlinkToDir(entry))
	})

	t.Run("file symlink", func(t *testing.T) {
		entry := findEntry("link-file")
		assert.False(t, rows.isSymlinkToDir(entry))
	})

	t.Run("broken symlink", func(t *testing.T) {
		entry := findEntry("broken-link")
		assert.False(t, rows.isSymlinkToDir(entry))
	})

	t.Run("non-symlink", func(t *testing.T) {
		entry := findEntry("target-dir")
		assert.False(t, rows.isSymlinkToDir(entry))
	})

	t.Run("non-file store", func(t *testing.T) {
		entry := findEntry("link-dir")
		remoteStore := mockStore{root: url.URL{Scheme: "ftp"}}
		remoteRows := NewFileRows(&DirContext{Store: remoteStore, Path: tmpDir})
		assert.False(t, remoteRows.isSymlinkToDir(entry))
	})
}
