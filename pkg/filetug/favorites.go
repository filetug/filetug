package filetug

import (
	"context"
	"net/url"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/ftpfile"
	"github.com/filetug/filetug/pkg/files/httpfile"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type favorites struct {
	*sneatv.Boxed
	flex  *tview.Flex
	nav   *Navigator
	list  *tview.List
	items []ftfav.Favorite
	prev  current
}

func (f *favorites) ShowFavorites() {
	f.prev = f.nav.current
	f.nav.left.SetContent(f)
	f.nav.setAppFocus(f.list)
}

func builtInFavorites() []ftfav.Favorite {
	testFtpServerUrl, _ := url.Parse("ftp://demo:password@test.rebex.net")
	return []ftfav.Favorite{
		{Store: url.URL{Scheme: "file"}, Path: "/", Shortcut: '/', Description: "root"},
		{Store: url.URL{Scheme: "file"}, Path: "~", Shortcut: 'h', Description: "User's home directory"},
		{Store: url.URL{Scheme: "file"}, Path: "~/Documents", Description: "Documents"},
		{Store: url.URL{Scheme: "file"}, Path: "~/projects", Description: "Projects"},
		{Store: url.URL{Scheme: "file"}, Path: "~/.filetug", Description: "FileTug settings dir"},
		{Store: url.URL{Scheme: "https", Host: "www.kernel.org", Path: "/pub/"}, Path: "/pub/", Description: "The Linux Kernel Archives"},
		{Store: *testFtpServerUrl, Description: "The Linux Kernel Archives"},
	}
}
func newFavorites(nav *Navigator) *favorites {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetTitle(" Favorites ")
	list := tview.NewList()
	list.SetSecondaryTextColor(tcell.ColorGray)
	footer := tview.NewTextView().SetText("<esc> to go back").SetTextColor(tcell.ColorGray)
	f := &favorites{
		flex:  flex,
		list:  list,
		nav:   nav,
		items: builtInFavorites(),
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(1, -1),
			sneatv.WithFooter(footer),
		),
	}
	f.flex.AddItem(f.list, 0, 1, true)

	//f.flex.AddItem(hint, 1, 0, false)
	f.setItems()
	f.list.SetInputCapture(f.inputCapture)
	f.list.SetChangedFunc(f.changed)
	return f
}

func (f *favorites) setItems() {
	f.list.Clear()
	i := 0
	for _, item := range f.items {
		if item.Store.String() == "" {
			item.Store.Scheme = "file"
		}
		if item.Store.Scheme != "file" || (item.Path != "~" && item.Path != "/") {
			i++
		}
		var mainText string

		if item.Store.Scheme == "file" {
			switch item.Path {
			case "/":
				mainText = "/ [darkgray::i] root"
			case "~":
				mainText = "~ [darkgray::i] User's home directory"
			default:
				mainText = item.Path
			}
		} else {
			storeURL := item.Store
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

func (f *favorites) selected(item ftfav.Favorite) {
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
		dirContext := files.NewDirContext(f.nav.store, f.prev.dir, nil)
		f.nav.goDir(dirContext)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.setAppFocus(f.nav.dirsTree)
		return nil
	case tcell.KeyLeft:
		f.nav.setAppFocus(f.nav.files.table)
		return nil
	case tcell.KeyUp, tcell.KeyDown:
		return event
	default:
		return event
	}
}

func (f *favorites) activateFavorite(item ftfav.Favorite, previewMode bool) {
	dirPath := f.setStore(item)
	if previewMode {
		ctx := context.Background()
		dirContext := files.NewDirContext(f.nav.store, dirPath, nil)
		f.nav.showDir(ctx, nil, dirContext, false)
	} else {
		dirContext := files.NewDirContext(f.nav.store, dirPath, nil)
		f.nav.goDir(dirContext)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.setAppFocus(f.nav.dirsTree)
	}
}

func (f *favorites) setStore(item ftfav.Favorite) (dirPath string) {
	dirPath = item.Path
	root := item.Store
	if storeRootUrl := f.nav.store.RootURL(); storeRootUrl.String() != root.String() {
		var store files.Store
		switch strings.ToLower(root.Scheme) {
		case "http", "https":
			store = httpfile.NewStore(root)
		case "ftp", "ftps":
			store = ftpfile.NewStore(root)
		case "file":
			if root.Path == "" {
				root.Path = "/"
			}
			store = osfile.NewStore(root.Path)
		}
		if store != nil {
			f.nav.SetStore(store)
		}
	}
	return
}
