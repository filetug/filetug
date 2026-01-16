package filetug

import (
	"net/url"
	"strings"

	"github.com/datatug/filetug/pkg/files/ftpfile"
	"github.com/datatug/filetug/pkg/files/httpfile"
	"github.com/datatug/filetug/pkg/files/osfile"
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
		{path: "~", shortcut: 'h', description: "User's home directory"},
		{path: "~/Documents", description: "Documents"},
		{path: "~/projects", description: "Projects"},
		{path: "~/.filetug", description: "FileTug settings dir"},
		{path: "https://www.kernel.org/pub/", description: "The Linux Kernel Archives"},
		{path: "ftp://demo:password@test.rebex.net/", description: "The Linux Kernel Archives"},
	}
}
func newFavorites(nav *Navigator) *favorites {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetTitle(" Favorites ")
	f := &favorites{
		Flex:  flex,
		list:  tview.NewList().SetSecondaryTextColor(tcell.ColorGray),
		nav:   nav,
		items: builtInFavorites(),
		boxed: newBoxed(
			flex,
			WithLeftBorder(1, -1),
		),
	}
	f.AddItem(f.list, 0, 1, true)
	hint := tview.NewTextView().SetText("<esc> to go back").SetTextColor(tcell.ColorGray)
	f.AddItem(hint, 1, 0, false)
	f.setItems()
	f.list.SetInputCapture(f.inputCapture)
	f.list.SetChangedFunc(f.changed)
	return f
}

func (f *favorites) changed(index int, _ string, _ string, _ rune) {
	item := f.items[index]
	dirPath := item.path
	if strings.HasPrefix(item.path, "https://") {
		root, _ := url.Parse(item.path)
		dirPath = root.Path
		f.nav.store = httpfile.NewStore(*root)
	} else if strings.HasPrefix(item.path, "ftp://") {
		u, _ := url.Parse(item.path)
		dirPath = u.Path
		password, _ := u.User.Password()
		f.nav.store = ftpfile.NewStore(u.Host, u.User.Username(), password)
	} else {
		switch f.nav.store.(type) {
		case *osfile.Store: // No change needed
		default:
			f.nav.store = osfile.NewStore("/")
		}
	}
	f.nav.goDir(dirPath)
}

func (f *favorites) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		f.nav.goDir(f.prev.dir)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.app.SetFocus(f.nav.dirsTree)
		return nil
	case tcell.KeyLeft:
		f.nav.app.SetFocus(f.nav.files.table)
		return nil
	default:
		return event
	}
}

func (f *favorites) setItems() {
	f.list.Clear()
	i := 0
	for _, item := range f.items {
		if item.path != "~" && item.path != "/" {
			i++
		}
		var mainText string
		switch item.path {
		case "/":
			mainText = "/ [darkgray::i] root"
		case "~":
			mainText = "~ [darkgray::i] User's home directory"
		default:
			mainText = item.path
		}

		//mainText += " - [::i]" + item.description + "[-:-:I]"
		secondText := item.description
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
