package filetug

import (
	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type favorite struct {
	path        string
	shortcut    rune
	description string
}

type favorites struct {
	*tview.Flex
	boxed *boxed
	nav   *Navigator
	list  *tview.List
	items []favorite
	prev  current
}

func (f *favorites) Draw(screen tcell.Screen) {
	f.boxed.Draw(screen)
}

func (f *favorites) ShowFavorites() {
	f.prev = f.nav.current
	f.nav.left.SetContent(f)
	f.nav.app.SetFocus(f.list)
}

func builtInFavorites() []favorite {
	return []favorite{
		{path: "/", shortcut: '/', description: "root"},
		{path: "~", shortcut: '0', description: "User's home directory"},
		{path: "~/Documents", description: "Documents"},
		{path: "~/projects", description: "Projects"},
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
		case tcell.KeyEscape:
			f.nav.goDir(f.prev.dir)
			f.nav.left.SetContent(f.nav.dirsTree)
			f.nav.app.SetFocus(f.nav.dirsTree)
			return nil
		case tcell.KeyLeft:
			f.nav.app.SetFocus(f.nav.files.Table.Table)
			return nil
		default:
			return event
		}
	})
	f.list.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		item := f.items[index]
		f.nav.goDir(item.path)
	})
	return f
}

func (f *favorites) setItems() {
	f.list.Clear()
	i := 0
	for _, item := range f.items {
		if item.path != "~" && item.path != "/" {
			i++
		}
		var mainText string
		if string(item.shortcut) != item.path {
			mainText = item.path
		}
		mainText += " - [::i]" + item.description + "[-:-:I]"
		var secondText string
		if item.path == "~" {
			secondText = fsutils.ExpandHome("~")
		}
		shortcut := item.shortcut
		if shortcut == 0 {
			shortcut = '0' + rune(i)
		}
		f.list.AddItem(mainText, secondText, shortcut, func() {
			f.selected(item)
		})
	}
}

func (f *favorites) selected(item favorite) {
	f.nav.goDir(item.path)
	f.nav.left.SetContent(f.nav.dirsTree)
	f.nav.app.SetFocus(f.nav.dirsTree)
}
