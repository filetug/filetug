package filetug

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type left struct {
	*tview.Flex
	nav *Navigator
}

func (l *left) onFocus() {
	l.nav.activeCol = 0
	l.SetBorderColor(Style.FocusedBorderColor)
}

func (l *left) onBlur() {
	l.SetBorderColor(Style.BlurBorderColor)
}

func createLeft(nav *Navigator) {
	nav.left = &left{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		nav:  nav,
	}
	//nav.left.SetBorder(true)
	//nav.left.AddItem(nav.favorites, 3, 0, false)
	nav.left.AddItem(nav.dirsTree, 0, 1, true)
	nav.left.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			nav.app.SetFocus(nav.files)
			return nil
		default:
			return event
		}
	})

	nav.left.SetFocusFunc(nav.left.onFocus)

	nav.left.SetBlurFunc(nav.left.onBlur)

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
