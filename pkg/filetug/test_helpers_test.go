package filetug

import "os"

type mockFileInfo struct {
	os.FileInfo
	size  int64
	isDir bool
}

func (m mockFileInfo) Size() int64 { return m.size }
func (m mockFileInfo) IsDir() bool { return m.isDir }
