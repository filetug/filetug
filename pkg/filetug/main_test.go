package filetug

import (
	"testing"

	"github.com/rivo/tview"
)

type setupApp struct {
	enableMouseCalled bool
	setRootCalled     bool
	setRootFullscreen bool
}

func (a *setupApp) Run() error { return nil }

func (a *setupApp) QueueUpdateDraw(f func()) {
	if f != nil {
		f()
	}
}

func (a *setupApp) SetFocus(p tview.Primitive) { _ = p }

func (a *setupApp) SetRoot(root tview.Primitive, fullscreen bool) {
	_, _ = root, fullscreen
	a.setRootCalled = true
	a.setRootFullscreen = fullscreen
}

func (a *setupApp) Stop() {}

func (a *setupApp) EnableMouse(b bool) { a.enableMouseCalled = b }

func TestSetupApp(t *testing.T) {
	app := &setupApp{}
	SetupApp(app)
	if !app.enableMouseCalled {
		t.Fatal("expected EnableMouse(true) to be called")
	}
	if !app.setRootCalled {
		t.Fatal("expected SetRoot to be called")
	}
	if !app.setRootFullscreen {
		t.Fatal("expected SetRoot to be called with fullscreen=true")
	}
}
