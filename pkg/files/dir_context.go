package files

import (
	"os"
	"path"
	"strings"
	"time"
)

var _ EntryWithDirPath = (*DirContext)(nil)

type DirContext struct {
	Store    Store
	Path     string
	children []os.DirEntry
	time     time.Time
}

func (d *DirContext) Timestamp() time.Time {
	return d.time
}

func (d *DirContext) SetChildren(entries []os.DirEntry) {
	d.time = time.Now()
	d.children = entries
}

func (d *DirContext) Entries() []EntryWithDirPath {
	entries := make([]EntryWithDirPath, len(d.children))
	for i, child := range d.children {
		entries[i] = NewEntryWithDirPath(child, d.Path)
	}
	return entries
}

func (d *DirContext) Children() []os.DirEntry {
	return d.children
}

func (d *DirContext) DirPath() string {
	if d.Path == "" {
		return ""
	}
	return path.Dir(d.Path)
}

func (d *DirContext) FullName() string {
	return d.Path
}

func (d *DirContext) String() string {
	return d.Path
}

func (d *DirContext) Name() string {
	if d.Path == "" {
		return ""
	}
	if d.Path == "/" {
		return "/"
	}
	trimmed := strings.TrimSuffix(d.Path, "/")
	return path.Base(trimmed)
}

func (d *DirContext) IsDir() bool {
	return true
}

func (d *DirContext) Type() os.FileMode {
	return os.ModeDir
}

func (d *DirContext) Info() (os.FileInfo, error) {
	if d.Path == "" {
		return nil, nil
	}
	if d.Store != nil && d.Store.RootURL().Scheme == "file" {
		return os.Stat(d.Path)
	}
	return nil, nil
}

func NewDirContext(store Store, path string, children []os.DirEntry) *DirContext {
	return &DirContext{
		Store:    store,
		Path:     path,
		children: children,
	}
}
