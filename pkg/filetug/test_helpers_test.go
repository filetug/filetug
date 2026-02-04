package filetug

import (
	"os"
	"sync"
	"testing"
)

var testGlobalLock sync.Mutex

func withTestGlobalLock(t *testing.T) {
	t.Helper()
	testGlobalLock.Lock()
	t.Cleanup(func() {
		testGlobalLock.Unlock()
	})
}

type mockFileInfo struct {
	os.FileInfo
	size  int64
	isDir bool
}

func (m mockFileInfo) Size() int64 { return m.size }
func (m mockFileInfo) IsDir() bool { return m.isDir }
