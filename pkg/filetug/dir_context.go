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

func newDirContext(store files.Store, path string, children []os.DirEntry) *DirContext {
	return &DirContext{
		Store:    store,
		Path:     path,
		children: children,
	}
}
