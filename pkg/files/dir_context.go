package files

import (
	"os"
	"path"
	"strings"
)

var _ EntryWithDirPath = (*DirContext)(nil)

type DirContext struct {
	Store    Store
	Path     string
	children []os.DirEntry
}

func (c *DirContext) SetChildren(entries []os.DirEntry) {
	c.children = entries
}

func (c *DirContext) Entries() []EntryWithDirPath {
	entries := make([]EntryWithDirPath, len(c.children))
	for i, child := range c.children {
		entries[i] = NewEntryWithDirPath(child, c.Path)
	}
	return entries
}

func (c *DirContext) Children() []os.DirEntry {
	return c.children
}

func (c *DirContext) DirPath() string {
	if c.Path == "" {
		return ""
	}
	return path.Dir(c.Path)
}

func (c *DirContext) FullName() string {
	return c.Path
}

func (c *DirContext) String() string {
	return c.Path
}

func (c *DirContext) Name() string {
	if c.Path == "" {
		return ""
	}
	if c.Path == "/" {
		return "/"
	}
	trimmed := strings.TrimSuffix(c.Path, "/")
	return path.Base(trimmed)
}

func (c *DirContext) IsDir() bool {
	return true
}

func (c *DirContext) Type() os.FileMode {
	return os.ModeDir
}

func (c *DirContext) Info() (os.FileInfo, error) {
	if c.Path == "" {
		return nil, nil
	}
	if c.Store != nil && c.Store.RootURL().Scheme == "file" {
		return os.Stat(c.Path)
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
