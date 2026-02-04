package filetug

import "github.com/rivo/tview"

// testApp is a minimal navigator.App implementation for tests that need
// a deterministic QueueUpdateDraw hook without gomock expectations.
type testApp struct {
	queueUpdateDraw func(f func())
}

func (a *testApp) Run() error { return nil }

func (a *testApp) QueueUpdateDraw(f func()) {
	if a.queueUpdateDraw != nil {
		a.queueUpdateDraw(f)
		return
	}
	if f != nil {
		f()
	}
}

func (a *testApp) SetFocus(p tview.Primitive) {
	_ = p
}

func (a *testApp) SetRoot(root tview.Primitive, fullscreen bool) {
	_, _ = root, fullscreen
}

func (a *testApp) Stop() {}

func (a *testApp) EnableMouse(_ bool) {}
