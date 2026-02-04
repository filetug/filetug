package viewers

import "sync"

var textPreviewerTestLock sync.Mutex

func withTextPreviewerTestLock(t testingT) {
	t.Helper()
	textPreviewerTestLock.Lock()
	t.Cleanup(func() {
		textPreviewerTestLock.Unlock()
	})
}

type testingT interface {
	Helper()
	Cleanup(func())
}
