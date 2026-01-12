package filetug

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestLeft(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))

	t.Run("LeftFocusBlur", func(t *testing.T) {
		nav.left.onFocus()
	})

	t.Run("DirsFocusBlur", func(t *testing.T) {
		nav.dirsFocusFunc()
		nav.dirsBlurFunc()
	})

	t.Run("LeftInputCapture", func(t *testing.T) {
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	})

	t.Run("DirsInputCapture", func(t *testing.T) {
		// Mock current node to avoid nil dereference in GetCurrentNode().GetReference()
		nav.dirsTree.SetCurrentNode(tview.NewTreeNode("test").SetReference("."))
		nav.dirsTree.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		nav.dirsTree.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

		// Test KeyUp at root
		nav.dirsTree.SetCurrentNode(nav.dirsTree.GetRoot())
		nav.dirsTree.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	})

	t.Run("TreeViewInputCapture_NoRef", func(t *testing.T) {
		nav.dirsTree.SetCurrentNode(tview.NewTreeNode("test"))
		nav.dirsTree.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})

	t.Run("NavigatorInputCapture_Enter", func(t *testing.T) {
		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})

	t.Run("Left_BlurFunc", func(t *testing.T) {
		nav.left.onBlur()
	})

	t.Run("Dirs_BlurFunc", func(t *testing.T) {
		nav.dirsBlurFunc()
	})
}
