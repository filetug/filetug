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

var _ files.EntryWithDirPath = (*DirContext)(nil)

type DirContext struct {
	files.EntryWithDirPath
	Store    files.Store
	Path     string
	children []os.DirEntry
}

func (c *DirContext) Entries() []files.EntryWithDirPath {
	entries := make([]files.EntryWithDirPath, len(c.children))
	for i, child := range c.children {
		entries[i] = files.NewEntryWithDirPath(child, c.Path)
	}
	return entries
}

func (c *DirContext) Children() []os.DirEntry {
	return c.children
}

func newDirContext(store files.Store, path string, children []os.DirEntry) *DirContext {
	return &DirContext{
		EntryWithDirPath: nil, // TODO: assign from an argument
		Store:            store,
		Path:             path,
		children:         children,
	}
}
