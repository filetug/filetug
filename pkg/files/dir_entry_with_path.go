package files

import (
	"os"
	"path"
	"path/filepath"
)

type EntryWithDirPath struct {
	os.DirEntry
	Dir string
}

func (c EntryWithDirPath) FullName() string {
	name := c.Name()
	return filepath.Join(c.Dir, name)
}

func (c EntryWithDirPath) String() string {
	name := c.Name()
	return path.Join(c.Dir, name)
}

func NewEntryWithDirPath(entry os.DirEntry, dir string) *EntryWithDirPath {
	return &EntryWithDirPath{
		Dir:      dir,
		DirEntry: entry,
	}
}
