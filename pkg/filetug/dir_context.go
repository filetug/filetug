package filetug

import (
	"os"

	"github.com/filetug/filetug/pkg/files"
)

type DirItem struct {
	Name  string
	IsDir bool
	Size  int64
}

type DirContext struct {
	Store    files.Store
	Path     string
	children []os.DirEntry
}

func (c *DirContext) Entries() []files.EntryWithDirPath {
	entries := make([]files.EntryWithDirPath, len(c.children))
	for i, child := range c.children {
		entries[i] = files.EntryWithDirPath{
			DirEntry: child,
			Dir:      c.Path,
		}
	}
	return entries
}

func (c *DirContext) Children() []os.DirEntry {
	return c.children
}

func newDirContext(store files.Store, path string, children []os.DirEntry) *DirContext {
	return &DirContext{
		Store:    store,
		Path:     path,
		children: children,
	}
}
