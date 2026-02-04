package filetug

import (
	"context"
	"fmt"
	"net/url"
	"os"
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

type favoritesPanel struct {
	*sneatv.Boxed
	flex           *tview.Flex
	nav            *Navigator
	list           *tview.List
	items          []ftfav.Favorite
	addContainer   *tview.Flex
	addFormVisible bool
	addButton      *tview.Button
}

var (
	addFavorite    = ftfav.AddFavorite
	deleteFavorite = ftfav.DeleteFavorite
	getFavorites   = ftfav.GetFavorites
)

func (nav *Navigator) ShowFavorites() {
	nav.prev = nav.current
	nav.left.SetContent(nav.favorites)
	nav.favorites.updateAddCurrentForm()
	nav.app.SetFocus(nav.favorites.list)
}

func builtInFavorites() []ftfav.Favorite {
	return []ftfav.Favorite{
		{Store: url.URL{Scheme: "file"}, Path: "/", Shortcut: '/', Description: "root"},
		{Store: url.URL{Scheme: "file"}, Path: "~", Shortcut: 'h', Description: "User's home directory"},
	}
}

func newFavoritesPanel(nav *Navigator) *favoritesPanel {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetTitle(" Favorites ")
	list := tview.NewList()
	list.SetSecondaryTextColor(tcell.ColorGray)
	footer := tview.NewTextView().SetText("<esc> to go back").SetTextColor(tcell.ColorGray)
	addButton := tview.NewButton("Add Current dir to favorites")
	addContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	addContainer.AddItem(addButton, 1, 0, false)
	f := &favoritesPanel{
		flex:         flex,
		list:         list,
		nav:          nav,
		items:        builtInFavorites(),
		addContainer: addContainer,
		addButton:    addButton,
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(1, -1),
			sneatv.WithFooter(footer),
		),
	}
	addButton.SetSelectedFunc(f.addCurrentFavorite)
	addButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if f.nav != nil && f.nav.app != nil {
				f.nav.app.SetFocus(f.list)
			}
			return nil
		default:
			return event
		}
	})
	f.flex.AddItem(f.list, 0, 1, true)

	//f.flex.AddItem(hint, 1, 0, false)
	f.setItems()
	f.updateAddCurrentForm()
	f.list.SetInputCapture(f.inputCapture)
	f.list.SetChangedFunc(f.changed)
	go func() {
		userFavorites, err := getFavorites()
		if err != nil {
			return
		}
		items := make([]ftfav.Favorite, 0, len(f.items)+len(userFavorites))
		items = append(items, f.items...)
		items = append(items, userFavorites...)
		update := func() {
			f.items = items
			f.setItems()
			f.updateAddCurrentForm()
		}
		if f.nav != nil && f.nav.app != nil {
			f.nav.app.QueueUpdateDraw(update)
		} else {
			update()
		}
	}()
	return f
}

func (f *favoritesPanel) setItems() {
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

func (f *favoritesPanel) selected(item ftfav.Favorite) {
	f.activateFavorite(item, false)
}

func (f *favoritesPanel) changed(index int, _ string, _ string, _ rune) {
	item := f.items[index]
	f.activateFavorite(item, true)
}

func (f *favoritesPanel) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	if f.nav == nil || f.nav.app == nil {
		return event
	}
	switch event.Key() {
	case tcell.KeyEnter:
		currentFav := f.items[f.list.GetCurrentItem()]
		f.activateFavorite(currentFav, false)
		return nil
	case tcell.KeyTab:
		if f.addFormVisible {
			f.nav.app.SetFocus(f.addButton)
			return nil
		}
		return event
	case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
		f.deleteCurrentFavorite()
		return nil
	case tcell.KeyEscape:
		if f.nav.prev.Dir() != nil {
			f.nav.goDir(f.nav.prev.Dir())
		}
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

func (f *favoritesPanel) updateAddCurrentForm() {
	showAddForm := f.shouldShowAddCurrentForm()
	if showAddForm == f.addFormVisible {
		return
	}
	if showAddForm {
		f.flex.AddItem(f.addContainer, 3, 0, false)
		f.addFormVisible = true
		return
	}
	f.flex.RemoveItem(f.addContainer)
	f.addFormVisible = false
	if f.nav != nil && f.nav.app != nil {
		f.nav.app.SetFocus(f.list)
	}
}

func (f *favoritesPanel) shouldShowAddCurrentForm() bool {
	currentFavorite, ok := f.currentFavorite()
	if !ok {
		return false
	}
	if f.hasFavorite(currentFavorite) {
		return false
	}
	return true
}

func (f *favoritesPanel) currentFavorite() (ftfav.Favorite, bool) {
	if f.nav == nil {
		return ftfav.Favorite{}, false
	}
	if f.nav.store == nil {
		return ftfav.Favorite{}, false
	}
	currentDirPath := f.nav.currentDirPath()
	if currentDirPath == "" {
		return ftfav.Favorite{}, false
	}
	storeURL := f.nav.store.RootURL()
	currentFavorite := ftfav.Favorite{
		Store: storeURL,
		Path:  currentDirPath,
	}
	return currentFavorite, true
}

func (f *favoritesPanel) hasFavorite(currentFavorite ftfav.Favorite) bool {
	currentKey := currentFavorite.Key()
	for _, item := range f.items {
		itemKey := item.Key()
		if itemKey == currentKey {
			return true
		}
	}
	return false
}

func (f *favoritesPanel) addCurrentFavorite() {
	currentFavorite, ok := f.currentFavorite()
	if !ok {
		return
	}
	err := addFavorite(currentFavorite)
	if err != nil {
		f.nav.showError(err)
		return
	}
	f.items = append(f.items, currentFavorite)
	f.setItems()
	f.updateAddCurrentForm()
}

func (f *favoritesPanel) deleteCurrentFavorite() {
	index := f.list.GetCurrentItem()
	if index < 0 || index >= len(f.items) {
		return
	}
	item := f.items[index]
	itemKey := item.Key()
	err := deleteFavorite(item)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "delete favorite failed: %v\n", err)
	}
	updated := make([]ftfav.Favorite, 0, len(f.items))
	for _, entry := range f.items {
		entryKey := entry.Key()
		if entryKey == itemKey {
			continue
		}
		updated = append(updated, entry)
	}
	f.items = updated
	f.setItems()
	f.updateAddCurrentForm()
}

func (f *favoritesPanel) activateFavorite(item ftfav.Favorite, previewMode bool) {
	if f.nav == nil || f.nav.store == nil {
		return
	}
	dirPath := f.setStore(item)
	if previewMode {
		ctx := context.Background()
		dirContext := files.NewDirContext(f.nav.store, dirPath, nil)
		f.nav.showDir(ctx, nil, dirContext, false)
	} else {
		dirContext := files.NewDirContext(f.nav.store, dirPath, nil)
		f.nav.goDir(dirContext)
		f.nav.left.SetContent(f.nav.dirsTree)
		f.nav.app.SetFocus(f.nav.dirsTree)
	}
}

func (f *favoritesPanel) setStore(item ftfav.Favorite) (dirPath string) {
	if f == nil || f.nav == nil || f.nav.store == nil {
		return ""
	}
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
