package tui

import (
	"net/url"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/ftpfile"
	"github.com/filetug/filetug/pkg/files/httpfile"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// favoritesID names the favorites list in its messages.
const favoritesID = "favorites"

// addItemID identifies the list row that adds the current directory.
const addItemID = "add"

// favoritesMsg is Alt+F: replace the tree with the favorites.
type favoritesMsg struct{}

// favoritesLoadedMsg carries the favorites saved by the user.
type favoritesLoadedMsg struct {
	Items []ftfav.Favorite
	Err   error
}

// Seams over the favorites file, replaced by tests.
var (
	addFavorite    = ftfav.AddFavorite
	deleteFavorite = ftfav.DeleteFavorite
	getFavorites   = ftfav.GetFavorites
)

// loadFavoritesCmd reads the saved favorites off the event loop.
func loadFavoritesCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := getFavorites()
		return favoritesLoadedMsg{Items: items, Err: err}
	}
}

// builtInFavorites are the favorites every user has.
func builtInFavorites() []ftfav.Favorite {
	return []ftfav.Favorite{
		{Store: url.URL{Scheme: "file"}, Path: "/", Shortcut: '/', Description: "root"},
		{Store: url.URL{Scheme: "file"}, Path: "~", Shortcut: 'h', Description: "User's home directory"},
	}
}

// favoritesScreen takes the place of the tree: a list of saved places. Moving
// the cursor previews a place, Enter goes there, Delete removes it, Esc goes
// back to the tree and the directory it showed.
type favoritesScreen struct {
	sess  *session
	list  widgets.List
	items []ftfav.Favorite
	// back is the tree this screen replaced; prevStore and prevDir are where the
	// user was.
	back      treeScreen
	prevStore files.Store
	prevDir   string
	keys      favoritesKeys
	w, h      int
}

type favoritesKeys struct{ Go, Remove, Back key.Binding }

func newFavoritesScreen(sess *session, back treeScreen) favoritesScreen {
	f := favoritesScreen{
		sess: sess, back: back, prevStore: sess.store, prevDir: sess.currentPath(),
		items: builtInFavorites(), list: widgets.NewList(favoritesID),
		keys: favoritesKeys{
			Go:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "go")),
			Remove: key.NewBinding(key.WithKeys("backspace", "delete"), key.WithHelp("del", "remove")),
			Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		},
	}
	f.setItems()
	return f
}

var (
	_ nav.Screen       = favoritesScreen{}
	_ nav.Titled       = favoritesScreen{}
	_ nav.ShortHelper  = favoritesScreen{}
	_ widgets.Boundary = favoritesScreen{}
)

// Init loads the saved favorites.
func (f favoritesScreen) Init() tea.Cmd { return loadFavoritesCmd() }

func (f favoritesScreen) Title() string { return "Favorites" }

func (f favoritesScreen) ShortHelp() []key.Binding {
	return []key.Binding{f.keys.Go, f.keys.Remove, f.keys.Back}
}

// AtEdge keeps Left for the file list, handled here, and Right for nothing.
func (f favoritesScreen) AtEdge(dir widgets.Direction) bool {
	if dir == widgets.Left {
		return false
	}
	return f.list.AtEdge(dir)
}

// favoriteText is the first line of a favorite in the list.
func favoriteText(item ftfav.Favorite) string {
	if item.Store.Scheme == "file" {
		switch item.Path {
		case "/":
			return "/ root"
		case "~":
			return "~ home"
		}
		return item.Path
	}
	store := item.Store
	store.User = nil
	scheme := store.Scheme
	store.Scheme = ""
	text := strings.TrimPrefix(strings.TrimPrefix(store.String(), "//"), "www.")
	return strings.ToUpper(scheme) + ": " + text
}

// favoriteDetail is the muted second line of a favorite.
func favoriteDetail(item ftfav.Favorite) string {
	if item.Path == "~" {
		return fsutils.ExpandHome("~")
	}
	return item.Description
}

// setItems fills the list from the favorites, with a row that adds the current
// directory when it is not a favorite yet.
func (f *favoritesScreen) setItems() {
	rows := make([]list.Item, 0, len(f.items)+1)
	digit := 0
	for i, item := range f.items {
		if item.Store.Scheme == "" {
			item.Store.Scheme = "file"
		}
		shortcut := item.Shortcut
		if shortcut == 0 {
			digit++
			shortcut = '0' + rune(digit)
		}
		rows = append(rows, widgets.MenuItem{
			ID: "fav", Label: favoriteText(item), Detail: favoriteDetail(item), Shortcut: shortcut, Ref: i,
		})
	}
	if f.canAddCurrent() {
		rows = append(rows, widgets.MenuItem{ID: addItemID, Label: "+ Add current dir to favorites"})
	}
	f.list.SetItems(rows...)
}

// currentFavorite is the directory the user was in when the favorites were
// opened, as a favorite. The directories previewed while the cursor moves are
// not it.
func (f favoritesScreen) currentFavorite() (ftfav.Favorite, bool) {
	if f.prevStore == nil || f.prevDir == "" {
		return ftfav.Favorite{}, false
	}
	return ftfav.Favorite{Store: f.prevStore.RootURL(), Path: f.prevDir}, true
}

// canAddCurrent reports whether the current directory is not a favorite yet.
func (f favoritesScreen) canAddCurrent() bool {
	current, ok := f.currentFavorite()
	if !ok {
		return false
	}
	for _, item := range f.items {
		if item.Key() == current.Key() {
			return false
		}
	}
	return true
}

// item is the favorite under the cursor, if the cursor is on one.
func (f favoritesScreen) item() (ftfav.Favorite, bool) {
	sel, ok := f.list.SelectedItem().(widgets.MenuItem)
	if !ok || sel.ID != "fav" {
		return ftfav.Favorite{}, false
	}
	return f.items[sel.Ref.(int)], true
}

func (f favoritesScreen) Update(msg tea.Msg) (nav.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.w, f.h = msg.Width, msg.Height
		f.list.SetSize(f.w, f.h)
	case nav.ScreenFocusMsg:
		if msg.Focused {
			f.list.Focus()
		} else {
			f.list.Blur()
		}
	case favoritesLoadedMsg:
		if msg.Err == nil {
			f.items = append(builtInFavorites(), msg.Items...)
			f.setItems()
		}
	case widgets.ItemHighlightedMsg:
		return f.highlighted(msg)
	case widgets.ItemSelectedMsg:
		return f.selected(msg)
	case tea.KeyPressMsg:
		return f.key(msg)
	default:
		var cmd tea.Cmd
		f.list, cmd = f.list.Update(msg)
		return f, cmd
	}
	return f, nil
}

// highlighted previews the place under the cursor.
func (f favoritesScreen) highlighted(msg widgets.ItemHighlightedMsg) (nav.Screen, tea.Cmd) {
	item, ok := f.item()
	if !ok || msg.ID != favoritesID {
		return f, nil
	}
	store, dir := f.storeFor(item)
	return f, widgets.Emit(showDirMsg{Store: store, Path: dir})
}

// selected goes to the place, or adds the current directory.
func (f favoritesScreen) selected(msg widgets.ItemSelectedMsg) (nav.Screen, tea.Cmd) {
	if msg.ID != favoritesID {
		return f, nil
	}
	if item, ok := f.item(); ok {
		store, dir := f.storeFor(item)
		return f, tea.Sequence(nav.SetPanels(f.back, nil, nav.FocusToMenu), widgets.Emit(goDirMsg{Store: store, Path: dir}))
	}
	return f.add()
}

// add saves the current directory as a favorite.
func (f favoritesScreen) add() (nav.Screen, tea.Cmd) {
	current, ok := f.currentFavorite()
	if !ok {
		return f, nil
	}
	if err := addFavorite(current); err != nil {
		return f, nav.Alert("Could not save the favorite", err.Error(), 0, nav.FocusToKeep)
	}
	f.items = append(slices.Clone(f.items), current)
	f.setItems()
	return f, nil
}

// remove deletes the favorite under the cursor from the file and the list.
func (f favoritesScreen) remove() (nav.Screen, tea.Cmd) {
	item, ok := f.item()
	if !ok {
		return f, nil
	}
	var cmd tea.Cmd
	if err := deleteFavorite(item); err != nil {
		cmd = nav.Alert("Could not remove the favorite", err.Error(), 0, nav.FocusToKeep)
	}
	kept := make([]ftfav.Favorite, 0, len(f.items))
	for _, it := range f.items {
		if it.Key() != item.Key() {
			kept = append(kept, it)
		}
	}
	f.items = kept
	f.setItems()
	return f, cmd
}

func (f favoritesScreen) key(msg tea.KeyPressMsg) (nav.Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, f.keys.Back):
		return f, tea.Sequence(
			nav.SetPanels(f.back, nil, nav.FocusToMenu),
			widgets.Emit(goDirMsg{Store: f.prevStore, Path: f.prevDir}),
		)
	case key.Matches(msg, f.keys.Remove):
		return f.remove()
	case msg.String() == "left":
		return f, nav.SetFocus(nav.FocusToContent)
	}
	var cmd tea.Cmd
	f.list, cmd = f.list.Update(msg)
	return f, cmd
}

func (f favoritesScreen) View() string { return f.list.View() }

// storeFor returns the store a favorite lives in and its directory. The current
// store is reused when the favorite is in it.
func (f favoritesScreen) storeFor(item ftfav.Favorite) (files.Store, string) {
	root := item.Store
	if f.sess.store != nil {
		current := f.sess.store.RootURL()
		if current.String() == root.String() {
			return f.sess.store, item.Path
		}
	}
	switch strings.ToLower(root.Scheme) {
	case "http", "https":
		return httpfile.NewStore(root), item.Path
	case "ftp", "ftps":
		if s := ftpfile.NewStore(root); s != nil {
			return s, item.Path
		}
	case "file":
		if root.Path == "" {
			root.Path = "/"
		}
		return osfile.NewStore(root.Path), item.Path
	}
	return f.sess.store, item.Path
}
