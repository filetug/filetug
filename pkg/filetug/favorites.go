package filetug

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type favorites struct {
	*tview.Flex
	boxed *boxed
	nav   *Navigator
	list  *tview.List
	items []favorite
}

type favorite struct {
	path        string
	description string
}

func (f *favorites) Draw(screen tcell.Screen) {
	f.boxed.Draw(screen)
}

func builtInFavorites() []favorite {
	return []favorite{
		{path: "~", description: "User's home directory"},
		{path: "/", description: "root"},
	}
}
func newFavorites(nav *Navigator) *favorites {
	flex := tview.NewFlex()
	flex.SetTitle(" Favorites ")
	f := &favorites{
		Flex:  flex,
		list:  tview.NewList(),
		nav:   nav,
		items: builtInFavorites(),
		boxed: newBoxed(
			flex,
			WithLeftBorder(1, -1),
		),
	}
	f.AddItem(f.list, 0, 1, true)
	f.setItems()
	f.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			f.nav.app.SetFocus(f.nav.files.Table.Table)
			return nil
		default:
			return event
		}
	})
	return f
}

func (f *favorites) setItems() {
	f.list.Clear()
	for _, item := range f.items {
		f.list.AddItem(item.path+" - [::i]"+item.description+"[-:-:I]", "", 0, func() {
			f.nav.goDir(item.path)
			f.nav.app.SetFocus(f.nav.files.Table.Table)
		})
	}
}
