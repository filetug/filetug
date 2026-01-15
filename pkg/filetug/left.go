package filetug

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func createLeft(nav *Navigator) {
	nav.left = newContainer(0, nav)
	nav.left.SetContent(nav.dirsTree)

	onLeftTreeViewFocus := func(t *tview.TreeView) {
		nav.activeCol = 0
		t.SetGraphicsColor(tcell.ColorWhite)
		nav.left.SetBorderColor(Style.FocusedBorderColor)
		if t.GetCurrentNode() == nil {
			children := t.GetRoot().GetChildren()
			if len(children) > 0 {
				t.SetCurrentNode(children[0])
			}
		}
	}

	onLeftTreeViewBlur := func(t *tview.TreeView) {
		t.SetGraphicsColor(Style.BlurGraphicsColor)
		nav.left.SetBorderColor(Style.BlurBorderColor)
	}

	nav.favorites.SetFocusFunc(func() {
		nav.activeCol = 0
	})
	nav.favoritesFocusFunc = func() {
		nav.activeCol = 0
	}
	nav.dirsTree.SetFocusFunc(func() {
		nav.activeCol = 0
		onLeftTreeViewFocus(nav.dirsTree.TreeView)
		nav.right.SetContent(nav.dirSummary)
	})
	nav.dirsFocusFunc = func() {
		nav.activeCol = 0
		onLeftTreeViewFocus(nav.dirsTree.TreeView)
	}
	nav.dirsTree.SetBlurFunc(func() {
		onLeftTreeViewBlur(nav.dirsTree.TreeView)
	})
	nav.dirsBlurFunc = func() {
		onLeftTreeViewBlur(nav.dirsTree.TreeView)
	}
}
