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
	l.nav.app.SetFocus(l.nav.favorites.TreeView)
}

func (l *left) onBlur() {
	l.SetBorderColor(Style.BlurBorderColor)
}

func createLeft(nav *Navigator) {
	nav.left = &left{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		nav:  nav,
	}
	nav.left.SetBorder(true)
	nav.left.AddItem(nav.favorites, 3, 0, false)
	nav.left.AddItem(nav.dirs, 0, 1, true)
	nav.left.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			nav.app.SetFocus(nav.files)
			return nil
		default:
			return event
		}
	})
	treeViewInputCapture := func(t *tview.TreeView, event *tcell.EventKey, f func(*tcell.EventKey) *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			ref := t.GetCurrentNode().GetReference()
			if ref != nil {
				dir := ref.(string)
				nav.goDir(dir)
				return nil
			}
		}
		if f != nil {
			return f(event)
		}
		return event
	}
	nav.favorites.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return treeViewInputCapture(nav.favorites.TreeView, event, func(key *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyUp:
				rootNode := nav.favorites.GetRoot()
				current := nav.favorites.GetCurrentNode()
				if current == rootNode || current == rootNode.GetChildren()[0] {
					nav.o.moveFocusUp(nav.favorites.TreeView)
					nav.favorites.SetCurrentNode(nil)
					return nil
				}
				return event
			case tcell.KeyDown:
				favNodes := nav.favorites.GetRoot().GetChildren()
				if nav.favorites.GetCurrentNode() == favNodes[len(favNodes)-1] {
					nav.favorites.SetCurrentNode(nil)
					nav.dirs.SetCurrentNode(nav.dirs.GetRoot())
					nav.app.SetFocus(nav.dirs.TreeView)
					return nil
				}
				return event
			default:
				return event
			}
		})
	})
	nav.dirs.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return treeViewInputCapture(nav.dirs.TreeView, event, func(key *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyUp && nav.dirs.GetCurrentNode() == nav.dirs.GetRoot() {
				children := nav.favorites.GetRoot().GetChildren()
				nav.favorites.SetCurrentNode(children[len(children)-1])
				nav.dirs.SetCurrentNode(nil)
				nav.app.SetFocus(nav.favorites.TreeView)
				return nil
			}
			return event
		})
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
		if nav.favorites.GetCurrentNode() == nil {
			nav.favorites.SetCurrentNode(nav.dirs.GetRoot().GetChildren()[0])
		}
		onLeftTreeViewFocus(nav.favorites.TreeView)
	})
	nav.favoritesFocusFunc = func() {
		nav.activeCol = 0
		if nav.favorites.GetCurrentNode() == nil {
			nav.favorites.SetCurrentNode(nav.dirs.GetRoot().GetChildren()[0])
		}
		onLeftTreeViewFocus(nav.favorites.TreeView)
	}
	nav.dirs.SetFocusFunc(func() {
		nav.activeCol = 0
		onLeftTreeViewFocus(nav.dirs.TreeView)
	})
	nav.dirsFocusFunc = func() {
		nav.activeCol = 0
		onLeftTreeViewFocus(nav.dirs.TreeView)
	}
	nav.favorites.SetBlurFunc(func() {
		onLeftTreeViewBlur(nav.favorites.TreeView)
	})
	nav.favoritesBlurFunc = func() {
		onLeftTreeViewBlur(nav.favorites.TreeView)
	}
	nav.dirs.SetBlurFunc(func() {
		onLeftTreeViewBlur(nav.dirs.TreeView)
	})
	nav.dirsBlurFunc = func() {
		onLeftTreeViewBlur(nav.dirs.TreeView)
	}
}
