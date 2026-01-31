package navigator

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		a := NewApp(nil)
		assert.NotNil(t, a)
	})
	t.Run("not_nil", func(t *testing.T) {
		app := tview.NewApplication()
		a := NewApp(app, WithQueueUpdateDraw(func(f func()) {
			f()
		}))
		assert.NotNil(t, a)

		ap := a.(*appProxy)
		assert.NotNil(t, ap.queueUpdateDraw)
		assert.NotNil(t, ap.setFocus)
		assert.NotNil(t, ap.setRoot)
		assert.NotNil(t, ap.enableMouse)
		assert.NotNil(t, ap.run)
		assert.NotNil(t, ap.stop)

		a.EnableMouse(true)
		root := tview.NewTextView()
		a.SetFocus(root)
		a.SetRoot(root, true)
		queueUpdateDrawCalled := false
		a.QueueUpdateDraw(func() {
			queueUpdateDrawCalled = true
		})
		assert.True(t, queueUpdateDrawCalled)
	})
}

func TestAppProxy_Methods(t *testing.T) {
	var (
		queueCalled bool
		focusCalled bool
		rootCalled  bool
		mouseCalled bool
		runCalled   bool
		stopCalled  bool
	)

	a := NewApp(nil,
		WithQueueUpdateDraw(func(f func()) { queueCalled = true; f() }),
		WithSetFocus(func(p tview.Primitive) { focusCalled = true }),
		WithSetRoot(func(root tview.Primitive, fullscreen bool) { rootCalled = true }),
		WithEnableMouse(func(b bool) { mouseCalled = true }),
		WithRun(func() error { runCalled = true; return nil }),
		WithStop(func() { stopCalled = true }),
	)

	t.Run("QueueUpdateDraw", func(t *testing.T) {
		innerCalled := false
		a.QueueUpdateDraw(func() { innerCalled = true })
		assert.True(t, queueCalled)
		assert.True(t, innerCalled)
	})

	t.Run("SetFocus", func(t *testing.T) {
		a.SetFocus(nil)
		assert.True(t, focusCalled)
	})

	t.Run("SetRoot", func(t *testing.T) {
		a.SetRoot(nil, true)
		assert.True(t, rootCalled)
	})

	t.Run("EnableMouse", func(t *testing.T) {
		a.EnableMouse(true)
		assert.True(t, mouseCalled)
	})

	t.Run("Run", func(t *testing.T) {
		err := a.Run()
		assert.NoError(t, err)
		assert.True(t, runCalled)
	})

	t.Run("Stop", func(t *testing.T) {
		a.Stop()
		assert.True(t, stopCalled)
	})
}
