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

	t.Run("FavoritesFocusBlur", func(t *testing.T) {
		nav.favoritesFocusFunc()
		nav.favoritesBlurFunc()
	})

	t.Run("DirsFocusBlur", func(t *testing.T) {
		nav.dirsFocusFunc()
		nav.dirsBlurFunc()
	})

	t.Run("LeftInputCapture", func(t *testing.T) {
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	})

	t.Run("FavoritesInputCapture", func(t *testing.T) {
		nav.favorites.SetCurrentNode(nav.favorites.GetRoot())
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))

		// Set current node to last child for KeyDown test
		children := nav.favorites.GetRoot().GetChildren()
		nav.favorites.SetCurrentNode(children[len(children)-1])
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	})

	t.Run("DirsInputCapture", func(t *testing.T) {
		// Mock current node to avoid nil dereference in GetCurrentNode().GetReference()
		nav.dirs.SetCurrentNode(tview.NewTreeNode("test").SetReference("."))
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

		// Test KeyUp at root
		nav.dirs.SetCurrentNode(nav.dirs.GetRoot())
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))

		// Test KeyDown at last favorite node
		favNodes := nav.favorites.GetRoot().GetChildren()
		nav.favorites.SetCurrentNode(favNodes[len(favNodes)-1])
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	})

	t.Run("TreeViewInputCapture_NoRef", func(t *testing.T) {
		nav.dirs.SetCurrentNode(tview.NewTreeNode("test"))
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})

	t.Run("onLeftTreeViewFocus_NoChildren", func(t *testing.T) {
		nav.favorites.GetRoot().SetChildren(nil)
		nav.favoritesFocusFunc()
	})

	t.Run("onLeftTreeViewFocus_WithChildren", func(t *testing.T) {
		nav.favorites.SetCurrentNode(nil)
		nav.favoritesFocusFunc()
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

	t.Run("Favorites_BlurFunc", func(t *testing.T) {
		nav.favoritesBlurFunc()
	})
}
