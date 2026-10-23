package tui

import (
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/tuigoff/tuigoff/pkg/grid"
)

// File list columns.
const (
	nameColumn = iota
	sizeColumn
	modifiedColumn
)

// fileColumns are the columns of the file list.
func fileColumns() []grid.Column {
	return []grid.Column{
		{Name: "Name", MaxWidth: 60},
		{Name: "Size", Numeric: true},
		{Name: "Modified"},
	}
}

// rowRef is what a file list row carries besides its text.
type rowRef struct {
	// Entry is the entry the row stands for.
	Entry files.EntryWithDirPath
	// IsDir is true for directories and symlinks to directories.
	IsDir bool
	// Parent marks the ".." row.
	Parent bool
	// Err is why the row's file information could not be read.
	Err error
}

// fileRows is the lazy row source of the file list: it formats only the rows
// of the page being drawn, so a directory of any size costs the same to show.
type fileRows struct {
	dir     *files.DirContext
	store   files.Store
	all     []files.EntryWithDirPath
	visible []files.EntryWithDirPath
	filter  ftui.Filter
	// hideParent hides the ".." row.
	hideParent bool
	marks      map[string]bool
	gitText    map[string]string
	// isSymlinkDir is a seam over the file system for symlinks to directories.
	isSymlinkDir func(fullName string) bool
	now          func() time.Time
}

// newFileRows creates the rows of a directory. showDirs makes directories part
// of the list; the tree shows them, so the list of the tree's root omits them.
func newFileRows(dir *files.DirContext, showDirs bool, marks map[string]bool) *fileRows {
	if dir == nil {
		dir = files.NewDirContext(nil, "", nil)
	}
	dirPath := dir.Path()
	if dirPath != "/" {
		dirPath = strings.TrimSuffix(dirPath, "/")
	}
	if dirPath != dir.Path() {
		dir = files.NewDirContext(dir.Store(), dirPath, dir.Children())
	}
	r := &fileRows{
		dir:          dir,
		store:        dir.Store(),
		all:          dir.Entries(),
		filter:       ftui.Filter{ShowDirs: showDirs},
		marks:        marks,
		gitText:      map[string]string{},
		isSymlinkDir: isSymlinkToDir,
		now:          time.Now,
	}
	r.hideParent = dirPath == "/"
	r.applyFilter()
	return r
}

// isSymlinkToDir reports whether a path is a symbolic link to a directory.
func isSymlinkToDir(fullName string) bool {
	info, err := os.Stat(fullName)
	return err == nil && info.IsDir()
}

// SetFilter replaces the filter and recomputes the visible entries.
func (r *fileRows) SetFilter(filter ftui.Filter) {
	r.filter = filter
	r.applyFilter()
}

func (r *fileRows) applyFilter() {
	r.visible = make([]files.EntryWithDirPath, 0, len(r.all))
	for _, entry := range r.all {
		if r.filter.IsVisible(entry) {
			r.visible = append(r.visible, entry)
		}
	}
}

// entryIsDir reports whether an entry is a directory or a symlink to one.
func (r *fileRows) entryIsDir(entry files.EntryWithDirPath) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&os.ModeSymlink == 0 {
		return false
	}
	if r.store == nil || r.store.RootURL().Scheme != "file" {
		return false
	}
	return r.isSymlinkDir(entry.FullName())
}

// offset is the number of rows before the first entry.
func (r *fileRows) offset() int {
	if r.hideParent {
		return 0
	}
	return 1
}

// Len implements grid.RowSource.
func (r *fileRows) Len() int { return len(r.visible) + r.offset() }

// IndexOf is the row of the entry called name, or -1.
func (r *fileRows) IndexOf(name string) int {
	for i, entry := range r.visible {
		if entry.Name() == name {
			return i + r.offset()
		}
	}
	return -1
}

// SetGitText records the git status text of an entry and reports whether it
// changed.
func (r *fileRows) SetGitText(fullPath, text string) bool {
	if r.gitText[fullPath] == text {
		return false
	}
	if text == "" {
		delete(r.gitText, fullPath)
	} else {
		r.gitText[fullPath] = text
	}
	return true
}

// parentEntry is the entry of the ".." row.
func (r *fileRows) parentEntry() files.EntryWithDirPath {
	if r.store == nil {
		return files.NewEntryWithDirPath(files.NewDirEntry("", true), "")
	}
	var parent string
	if r.dir.Path() == "~" {
		parent = fsutils.ExpandHome("~")
	} else {
		parent, _ = path.Split(r.dir.Path())
	}
	if parent != "/" {
		parent = strings.TrimSuffix(parent, "/")
	}
	parentDir, parentName := path.Split(parent)
	return files.NewEntryWithDirPath(files.NewDirEntry(parentName, true), parentDir)
}

// Entry returns the entry of row i, or nil for a row out of range.
func (r *fileRows) Entry(i int) files.EntryWithDirPath {
	if i < 0 || i >= r.Len() {
		return nil
	}
	if i == 0 && !r.hideParent {
		return r.parentEntry()
	}
	return r.visible[i-r.offset()]
}

// Row implements grid.RowSource.
func (r *fileRows) Row(i int) grid.Row {
	if i == 0 && !r.hideParent {
		text := ".."
		if r.store != nil && r.dir.Path() == r.store.RootURL().Path {
			text = "."
		}
		return grid.Row{Key: "..", Values: []any{text, "", ""}, Ref: rowRef{Entry: r.parentEntry(), IsDir: true, Parent: true}}
	}
	entry := r.visible[i-r.offset()]
	ref := rowRef{Entry: entry, IsDir: r.entryIsDir(entry)}
	name := r.nameText(entry, ref.IsDir)
	size, modified := "", ""
	info, err := entry.Info()
	switch {
	case err != nil:
		ref.Err = err
		modified = err.Error()
	case !isNil(info):
		if !entry.IsDir() {
			size = fsutils.GetSizeShortText(info.Size())
		}
		modified = r.modifiedText(info.ModTime())
	}
	return grid.Row{Key: entry.Name(), Values: []any{name, size, modified}, Ref: ref}
}

// isNil reports whether an interface holds a nil value.
func isNil(info os.FileInfo) bool {
	if info == nil {
		return true
	}
	v := reflect.ValueOf(info)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// nameText is the text of the name cell: selection mark, icon, name and git
// status.
func (r *fileRows) nameText(entry files.EntryWithDirPath, isDir bool) string {
	mark := " "
	if r.marks[entry.FullName()] {
		mark = "✓"
	}
	icon := "📄 "
	if isDir {
		icon = dirEmoji + " "
	}
	text := mark + icon + entry.Name()
	if status := r.gitText[entry.FullName()]; status != "" {
		text += " " + status
	}
	return text
}

// modifiedText is the date of a modification, or its time when it lies in the
// future.
func (r *fileRows) modifiedText(modTime time.Time) string {
	if modTime.After(r.now().Add(24 * time.Hour)) {
		return modTime.Format("15:04:05")
	}
	return modTime.Format("2006-01-02")
}
