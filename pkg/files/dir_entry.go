package files

import (
	"os"
	"path/filepath"
)

type DirEntryOption func(*DirEntry)

func NewDirEntry(name string, isDir bool, o ...FileInfoOption) DirEntry {
	if parent, _ := filepath.Split(name); parent != "" {
		// It's OK to have panic here.
		panic("dirPath entry name can not have path: " + name)
	}
	dirEntry := DirEntry{
		name:  name,
		isDir: isDir,
	}
	if len(o) > 0 {
		dirEntry.info = NewFileInfo(dirEntry, o...)
	}
	return dirEntry
}

var _ os.DirEntry = (*DirEntry)(nil)

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
	if d.info == nil {
		return nil, nil
	}
	return d.info, nil
}
