package filetug

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestOnMoveFocusUp(t *testing.T) {
	var s tview.Primitive
	f := func(source tview.Primitive) {
		s = source
	}
	o := OnMoveFocusUp(f)
	var options navigatorOptions
	o(&options)
	assert.Equal(t, f, options.moveFocusUp)

	textView := tview.NewTextView()
	options.moveFocusUp(textView)
	assert.Equal[tview.Primitive](t, textView, s)
}

func TestNavigator(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	if nav == nil {
		t.Fatal("nav is nil")
	}

	t.Run("SetFocus", func(t *testing.T) {
		nav.SetFocus()
	})

	t.Run("NavigatorInputCapture", func(t *testing.T) {
		altKey := func(r rune) *tcell.EventKey {
			return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModAlt)
		}
		nav.GetInputCapture()(altKey('0'))
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.activeCol = 1
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.activeCol = 2
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.GetInputCapture()(altKey('r'))
		nav.GetInputCapture()(altKey('h'))
		nav.GetInputCapture()(altKey('?'))

		// Test moveFocusUp in navigator
		nav.o.moveFocusUp(nav.files)

		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})
}

func TestNavigator_goDir(t *testing.T) {
	saveCurrentDir = func(string, string) {}
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))

	t.Run("goDir_Success", func(t *testing.T) {
		nav.goDir(".")
	})

	t.Run("goDir_NonExistent", func(t *testing.T) {
		nav.goDir("/non-existent-path-12345")
	})
}
