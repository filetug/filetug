package filetug

import (
	"context"
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
	Store       string `json:"Store,omitempty" yaml:"Store,omitempty"`
	Path        string `json:"path" yaml:"path"`
	Shortcut    rune   `json:"Shortcut" yaml:"Shortcut"`
	Description string `json:"Description" yaml:"Description"`
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
		{Store: "file:", Path: "/", Shortcut: '/', Description: "root"},
		{Store: "file:", Path: "~", Shortcut: 'h', Description: "User's home directory"},
		{Store: "file:", Path: "~/Documents", Description: "Documents"},
		{Store: "file:", Path: "~/projects", Description: "Projects"},
		{Store: "file:", Path: "~/.filetug", Description: "FileTug settings dir"},
		{Store: "https://www.kernel.org/pub/", Path: "/pub/", Description: "The Linux Kernel Archives"},
		{Store: "ftp://demo:password@test.rebex.net", Description: "The Linux Kernel Archives"},
	}
}
func newFavorites(nav *Navigator) *favorites {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetTitle(" Favorites ")
	list := tview.NewList()
	list.SetSecondaryTextColor(tcell.ColorGray)
	f := &favorites{
		Flex:  flex,
		list:  list,
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

func (f *favorites) setItems() {
	f.list.Clear()
	i := 0
	for _, item := range f.items {
		if item.Store == "" {
			item.Store = "file:"
		}
		if !strings.HasPrefix(item.Store, "file:") || (item.Path != "~" && item.Path != "/") {
			i++
		}
		var mainText string

		if strings.HasPrefix(item.Store, "file:") {
			switch item.Path {
			case "/":
				mainText = "/ [darkgray::i] root"
			case "~":
				mainText = "~ [darkgray::i] User's home directory"
			default:
				mainText = item.Path
			}
		} else {
			storeURL, _ := url.Parse(item.Store)
			storeURL.User = nil
			scheme := storeURL.Scheme
			storeURL.Scheme = ""
			mainText = storeURL.String()
			mainText = strings.TrimPrefix(mainText, "//")
			mainText = strings.TrimPrefix(mainText, "www.")
			mainText = strings.ToUpper(scheme) + ": " + mainText
		}

		//mainText += " - [::i]" + item.Description + "[-:-:I]"
		secondText := item.Description
		if item.Path == "~" {
			secondText = fsutils.ExpandHome("~")
		}
		shortcut := item.Shortcut
		if shortcut == 0 {
			shortcut = '0' + rune(i)
		}
		secondText = tview.Escape(secondText)
		f.list.AddItem(mainText, secondText, shortcut, func() {
			f.selected(item)
		})
	}
}

func (f *favorites) selected(item favorite) {
	f.activateFavorite(item, false)
}

func (f *favorites) changed(index int, _ string, _ string, _ rune) {
	item := f.items[index]
	f.activateFavorite(item, true)
}

func (f *favorites) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		currentFav := f.items[f.list.GetCurrentItem()]
		f.activateFavorite(currentFav, false)
		return nil
	case tcell.KeyEscape:
		f.nav.goDir(f.prev.dir)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.app.SetFocus(f.nav.dirsTree)
		return nil
	case tcell.KeyLeft:
		f.nav.app.SetFocus(f.nav.files.table)
		return nil
	case tcell.KeyUp, tcell.KeyDown:
		return event
	default:
		return event
	}
}

func (f *favorites) activateFavorite(item favorite, previewMode bool) {
	dirPath := f.setStore(item)
	if previewMode {
		ctx := context.Background()
		f.nav.showDir(ctx, nil, dirPath, false)
	} else {
		f.nav.goDir(dirPath)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.app.SetFocus(f.nav.dirsTree)
	}
}

func (f *favorites) setStore(item favorite) (dirPath string) {
	dirPath = item.Path
	root, err := url.Parse(item.Store)
	if err != nil {
		panic(err)
	}
	if storeRootUrl := f.nav.store.RootURL(); storeRootUrl.String() != root.String() {
		switch root.Scheme {
		case "http", "https":
			f.nav.store = httpfile.NewStore(*root)
		case "ftp", "ftps":
			f.nav.store = ftpfile.NewStore(*root)
		case "file":
			if root.Path == "" {
				root.Path = "/"
			}
			f.nav.store = osfile.NewStore(root.Path)
		}
	}
	return
}
