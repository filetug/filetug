package tui

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/tuigoff/tuigoff/pkg/grid"
)

// failingEntry is an os.DirEntry whose Info fails.
type failingEntry struct{ files.DirEntry }

func (failingEntry) Info() (fs.FileInfo, error) { return nil, errors.New("no info") }

// symlinkEntry is an os.DirEntry that is a symbolic link.
type symlinkEntry struct{ files.DirEntry }

func (symlinkEntry) Type() fs.FileMode { return fs.ModeSymlink }

// nilInfoEntry returns a typed nil *FileInfo from Info.
type nilInfoEntry struct{ files.DirEntry }

func (nilInfoEntry) Info() (fs.FileInfo, error) { var fi *files.FileInfo; return fi, nil }

func dirOf(store files.Store, dirPath string, children ...os.DirEntry) *files.DirContext {
	return files.NewDirContext(store, dirPath, children)
}

func TestNewFileRowsDefaults(t *testing.T) {
	r := newFileRows(nil, false, nil)
	if r.Len() != 1 || r.Entry(0) == nil {
		t.Fatalf("an empty directory still has its parent row: len=%d", r.Len())
	}
	r = newFileRows(dirOf(nil, "/a/b/"), false, nil)
	if r.dir.Path() != "/a/b" {
		t.Fatalf("the trailing slash is dropped: %q", r.dir.Path())
	}
	if r = newFileRows(dirOf(nil, "/"), false, nil); !r.hideParent || r.Len() != 0 {
		t.Fatal("the root directory has no parent row")
	}
}

func TestFileRowsRows(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	store := fakeStore{root: url.URL{Scheme: "file"}}
	children := []os.DirEntry{
		files.NewDirEntry("sub", true),
		files.NewDirEntry("a.go", false, files.Size(2048), files.ModTime(now)),
		files.NewDirEntry("future.txt", false, files.Size(1), files.ModTime(now.Add(48*time.Hour))),
		failingEntry{files.NewDirEntry("bad", false)},
		nilInfoEntry{files.NewDirEntry("nil", false)},
	}
	marks := map[string]bool{"/d/a.go": true}
	r := newFileRows(dirOf(store, "/d", children...), true, marks)
	r.now = func() time.Time { return now }
	if r.Len() != 6 {
		t.Fatalf("parent row and five entries: %d", r.Len())
	}

	parent := r.Row(0)
	if parent.Values[0] != ".." || !parent.Ref.(rowRef).Parent {
		t.Fatalf("parent row: %+v", parent)
	}
	dir := r.Row(1)
	if dir.Values[0] != " 📁 sub" || dir.Values[1] != "" || !dir.Ref.(rowRef).IsDir {
		t.Fatalf("directory row: %+v", dir)
	}
	file := r.Row(2)
	if file.Values[0] != "✓📄 a.go" || file.Values[1] != "2KB" || file.Values[2] != "2026-09-30" {
		t.Fatalf("file row: %+v", file)
	}
	if future := r.Row(3); future.Values[2] != "12:00:00" {
		t.Fatalf("a modification in the future shows its time: %+v", future.Values)
	}
	bad := r.Row(4)
	if bad.Values[2] != "no info" || bad.Ref.(rowRef).Err == nil {
		t.Fatalf("a failing entry shows the error: %+v", bad)
	}
	if nilInfo := r.Row(5); nilInfo.Values[1] != "" || nilInfo.Values[2] != "" {
		t.Fatalf("an entry without information has no size: %+v", nilInfo.Values)
	}

	if r.Entry(-1) != nil || r.Entry(6) != nil || r.Entry(2).Name() != "a.go" {
		t.Fatal("Entry looks rows up and rejects out of range")
	}
	if r.IndexOf("a.go") != 2 || r.IndexOf("nope") != -1 {
		t.Fatal("IndexOf finds the row of an entry")
	}
}

func TestFileRowsParentRowText(t *testing.T) {
	store := fakeStore{root: url.URL{Scheme: "file", Path: "/root"}}
	if got := newFileRows(dirOf(store, "/root"), false, nil).Row(0).Values[0]; got != "." {
		t.Fatalf("the store root has no parent, only itself: %v", got)
	}
	if got := newFileRows(dirOf(store, "/root/x"), false, nil).Row(0).Values[0]; got != ".." {
		t.Fatalf("parent row: %v", got)
	}
	if got := newFileRows(dirOf(nil, "/root/x"), false, nil).Row(0).Values[0]; got != ".." {
		t.Fatalf("parent row without a store: %v", got)
	}
}

func TestFileRowsParentEntry(t *testing.T) {
	home, _ := os.UserHomeDir()
	store := fakeStore{root: url.URL{Scheme: "file"}}
	cases := []struct {
		store files.Store
		dir   string
		want  string
	}{
		{nil, "/a/b", ""},
		{store, "/a/b", "/a"},
		{store, "/a", "/"},
		{store, "~", home},
	}
	for _, c := range cases {
		got := newFileRows(dirOf(c.store, c.dir), false, nil).parentEntry()
		full := got.FullName()
		if c.want == "" {
			if full != "" {
				t.Errorf("%s: %q", c.dir, full)
			}
			continue
		}
		if full != c.want {
			t.Errorf("%s: parent %q, want %q", c.dir, full, c.want)
		}
	}
}

func TestFileRowsFilterAndGitText(t *testing.T) {
	children := []os.DirEntry{files.NewDirEntry("sub", true), files.NewDirEntry(".dot", false), files.NewDirEntry("a.go", false)}
	r := newFileRows(dirOf(nil, "/d", children...), false, nil)
	if r.Len() != 2 { // the parent row and a.go
		t.Fatalf("hidden files and directories are filtered: %d", r.Len())
	}
	r.SetFilter(ftui.Filter{ShowHidden: true, ShowDirs: true})
	if r.Len() != 4 {
		t.Fatalf("everything shows: %d", r.Len())
	}

	full := "/d/a.go"
	if !r.SetGitText(full, "M") || r.SetGitText(full, "M") {
		t.Fatal("SetGitText reports changes only")
	}
	if got := r.nameText(files.NewEntryWithDirPath(files.NewDirEntry("a.go", false), "/d"), false); got != " 📄 a.go M" {
		t.Fatalf("git status follows the name: %q", got)
	}
	if !r.SetGitText(full, "") || r.SetGitText(full, "") {
		t.Fatal("empty text clears the status")
	}
}

func TestEntryIsDir(t *testing.T) {
	store := fakeStore{root: url.URL{Scheme: "file"}}
	r := newFileRows(dirOf(store, "/d"), false, nil)
	r.isSymlinkDir = func(string) bool { return true }
	link := files.NewEntryWithDirPath(symlinkEntry{files.NewDirEntry("l", false)}, "/d")
	plain := files.NewEntryWithDirPath(files.NewDirEntry("f", false), "/d")
	if !r.entryIsDir(link) || r.entryIsDir(plain) {
		t.Fatal("only symlinks are resolved")
	}
	r.store = fakeStore{root: url.URL{Scheme: "ftp"}}
	if r.entryIsDir(link) {
		t.Fatal("symlinks are only resolved on the local file system")
	}
	r.store = nil
	if r.entryIsDir(link) {
		t.Fatal("no store, no symlinks")
	}
}

func TestIsSymlinkToDir(t *testing.T) {
	dir := t.TempDir()
	if !isSymlinkToDir(dir) || isSymlinkToDir(dir+"/missing") {
		t.Fatal("a directory is one, a missing path is not")
	}
	write(t, dir+"/f", "x")
	if isSymlinkToDir(dir + "/f") {
		t.Fatal("a file is not")
	}
}

func TestIsNil(t *testing.T) {
	if !isNil(nil) {
		t.Fatal("a nil interface")
	}
	var fi *files.FileInfo
	if !isNil(fi) {
		t.Fatal("a typed nil")
	}
	if isNil(files.NewFileInfo(files.NewDirEntry("a", false))) {
		t.Fatal("a value")
	}
}

func TestFileColumns(t *testing.T) {
	cols := fileColumns()
	if len(cols) != 3 || cols[nameColumn].Name != "Name" || !cols[sizeColumn].Numeric || cols[modifiedColumn].Name != "Modified" {
		t.Fatalf("columns: %+v", cols)
	}
	var _ grid.RowSource = (*fileRows)(nil)
}
