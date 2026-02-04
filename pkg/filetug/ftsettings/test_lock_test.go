package ftsettings

import "sync"

var ftsettingsTestLock sync.Mutex

func withTestGlobalLock(t interface {
	Helper()
	Cleanup(func())
}) {
	t.Helper()
	ftsettingsTestLock.Lock()
	t.Cleanup(func() {
		ftsettingsTestLock.Unlock()
	})
}
