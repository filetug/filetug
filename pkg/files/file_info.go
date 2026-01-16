package files

import (
	"os"
	"time"
)

type FileInfoOption func(*FileInfo)

type FileInfo struct {
	DirEntry
	size    int64
	modTime time.Time
	sys     any
}

func NewFileInfo(dirEntry DirEntry, o ...FileInfoOption) (info *FileInfo) {
	info = &FileInfo{
		DirEntry: dirEntry,
	}
	for _, opt := range o {
		opt(info)
	}
	return
}

func Size(v int64) FileInfoOption {
	return func(info *FileInfo) {
		info.size = v
	}
}

func ModTime(v time.Time) FileInfoOption {
	return func(info *FileInfo) {
		info.modTime = v
	}
}

func (f *FileInfo) Name() string {
	if f == nil {
		return ""
	}
	return f.name
}
func (f *FileInfo) Size() int64 {
	if f == nil {
		return 0
	}
	return f.size
}
func (f *FileInfo) Mode() os.FileMode {
	if f == nil {
		return 0
	}
	return f.Type()
}
func (f *FileInfo) ModTime() time.Time {
	if f == nil {
		return time.Time{}
	}
	return f.modTime
}
func (f *FileInfo) IsDir() bool {
	if f == nil {
		return false
	}
	return f.isDir
}
func (f *FileInfo) Sys() any {
	if f == nil {
		return nil
	}
	return f.sys
}
