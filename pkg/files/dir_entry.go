package files

import (
	"os"
)

type DirEntryOption func(*DirEntry)

func NewDirEntry(name string, isDir bool, o ...FileInfoOption) DirEntry {
	dirEntry := DirEntry{
		name:  name,
		isDir: isDir,
	}
	if len(o) > 0 {
		dirEntry.info = NewFileInfo(dirEntry, o...)
	}
	return dirEntry
}

type DirEntry struct {
	name  string
	isDir bool
	info  *FileInfo
}

func (d DirEntry) Name() string { return d.name }
func (d DirEntry) IsDir() bool  { return d.isDir }
func (d DirEntry) Type() os.FileMode {
	if d.isDir {
		return os.ModeDir
	}
	return 0
}
func (d DirEntry) Info() (os.FileInfo, error) {
	return d.info, nil
}
