package filetug

import (
	"testing"

	"github.com/alecthomas/assert/v2"
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
