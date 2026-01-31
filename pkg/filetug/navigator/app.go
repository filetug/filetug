package navigator

import (
	"github.com/rivo/tview"
)

type App interface {
	Run() error
	QueueUpdateDraw(f func())
	SetFocus(p tview.Primitive)
	SetRoot(root tview.Primitive, fullscreen bool)
	Stop()
	EnableMouse(bool)
}

type AppMethod func(na *appProxy)

func NewApp(app *tview.Application, o ...AppMethod) App {
	a := &appProxy{}
	if app != nil {
		a.setFocus = func(primitive tview.Primitive) {
			_ = app.SetFocus(primitive)
		}
		a.setRoot = func(root tview.Primitive, fullscreen bool) {
			_ = app.SetRoot(root, fullscreen)
		}
		a.enableMouse = func(b bool) {
			_ = app.EnableMouse(b)
		}
		a.queueUpdateDraw = app.QueueUpdateDraw
		a.run = app.Run
		a.stop = app.Stop
	}
	for _, m := range o {
		m(a)
	}
	return a
}

func WithQueueUpdateDraw(queueUpdateDraw UpdateDrawQueuer) AppMethod {
	return func(na *appProxy) {
		na.queueUpdateDraw = func(f func()) *tview.Application {
			queueUpdateDraw(f)
			return nil
		}
	}
}

func WithSetFocus(setFocus Focuser) AppMethod {
	return func(na *appProxy) {
		na.setFocus = setFocus
	}
}

func WithSetRoot(setRoot RootSetter) AppMethod {
	return func(na *appProxy) {
		na.setRoot = setRoot
	}
}

func WithEnableMouse(enableMouse func(bool)) AppMethod {
	return func(na *appProxy) {
		na.enableMouse = enableMouse
	}
}

func WithRun(run func() error) AppMethod {
	return func(na *appProxy) {
		na.run = run
	}
}

func WithStop(stop func()) AppMethod {
	return func(na *appProxy) {
		na.stop = stop
	}
}

var _ App = (*appProxy)(nil)

type appProxy struct {
	queueUpdateDraw func(func()) *tview.Application
	setFocus        Focuser
	setRoot         RootSetter
	enableMouse     func(bool)
	run             func() error
	stop            func()
}

func (n appProxy) EnableMouse(b bool) {
	n.enableMouse(b)
}

func (n appProxy) QueueUpdateDraw(f func()) {
	n.queueUpdateDraw(f)
}

func (n appProxy) SetFocus(p tview.Primitive) {
	n.setFocus(p)
}

func (n appProxy) SetRoot(root tview.Primitive, fullscreen bool) {
	n.setRoot(root, fullscreen)
}

func (n appProxy) Run() error {
	return n.run()
}

func (n appProxy) Stop() {
	n.stop()
}
