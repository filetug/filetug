package files

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type EntryWithDirPath interface {
	os.DirEntry
	fmt.Stringer
	DirPath() string
	FullName() string
}

type entryWithDirPath struct {
	os.DirEntry
	dirPath string
}

func (c entryWithDirPath) DirPath() string {
	return c.dirPath
}

func (c entryWithDirPath) FullName() string {
	name := c.Name()
	return filepath.Join(c.dirPath, name)
}

func (c entryWithDirPath) String() string {
	name := c.Name()
	return path.Join(c.dirPath, name)
}

func NewEntryWithDirPath(entry os.DirEntry, dirPath string) EntryWithDirPath {
	name := entry.Name()
	if name != "/" {
		if namePath, _ := path.Split(name); namePath != "" {
			panic("entry name should have no path: " + name)
		}
	}
	return &entryWithDirPath{
		dirPath:  dirPath,
		DirEntry: entry,
	}
}
