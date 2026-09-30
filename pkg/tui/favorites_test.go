package tui

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/nav/navtest"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func favoritesOf(h *navtest.Harness) (favoritesScreen, bool) {
	f, ok := h.Model().Menu().(favoritesScreen)
	return f, ok
}

// withFavorites makes the saved favorites what the test wants.
func withFavorites(t *testing.T, saved ...ftfav.Favorite) *[]ftfav.Favorite {
	t.Helper()
	isolateState(t)
	store := saved
	getFavorites = func() ([]ftfav.Favorite, error) { return store, nil }
	addFavorite = func(f ftfav.Favorite) error { store = append(store, f); return nil }
	deleteFavorite = func(f ftfav.Favorite) error {
		var kept []ftfav.Favorite
		for _, it := range store {
			if it.Key() != f.Key() {
				kept = append(kept, it)
			}
		}
		store = kept
		return nil
	}
	return &store
}

func openWithFavorites(t *testing.T, dir string, saved ...ftfav.Favorite) (*navtest.Harness, *[]ftfav.Favorite) {
	t.Helper()
	store := withFavorites(t, saved...)
	h := navtest.New(t, rootPage(Options{Path: dir}), navtest.WithNav(shellOptions()...), navtest.WithSize(140, 30))
	return h, store
}

func TestFavoritesReplaceTheTreeAndGoBack(t *testing.T) {
	dir := tree(t)
	h, _ := openWithFavorites(t, dir)
	h.Press("alt+f")
	h.RequireContains("Favorites").RequireContains("/ root").RequireContains("~ home").RequireContains("+ Add current dir")
	if _, ok := favoritesOf(h); !ok || h.Model().Zone() != nav.FocusToMenu {
		t.Fatal("the favorites take the tree's place")
	}
	h.Press("down", "down") // cursor moves: home is previewed, then the add row
	h.Press("esc")
	if _, ok := favoritesOf(h); ok {
		t.Fatal("Esc brings the tree back")
	}
	if got := browserOf(h).sess.treeRoot.Path(); got != dir {
		t.Fatalf("and the directory the user was in: %q", got)
	}
	h.RequireContains("📁 alpha")
}

func TestFavoritesPreviewAndGo(t *testing.T) {
	dir := tree(t)
	h, _ := openWithFavorites(t, dir)
	h.Press("alt+f", "down") // home
	home, _ := os.UserHomeDir()
	if got := browserOf(h).sess.currentPath(); got != home {
		t.Fatalf("moving the cursor previews the place: %q", got)
	}
	h.Press("enter")
	if _, ok := favoritesOf(h); ok {
		t.Fatal("Enter goes there and brings the tree back")
	}
	if got := browserOf(h).sess.treeRoot.Path(); got != home {
		t.Fatalf("the home directory is the root: %q", got)
	}
}

func TestFavoritesAddTheDirectoryTheUserCameFrom(t *testing.T) {
	dir := tree(t)
	h, saved := openWithFavorites(t, dir)
	h.Press("alt+f", "down", "down", "enter") // the add row, after previewing home
	if len(*saved) != 1 || (*saved)[0].Path != dir {
		t.Fatalf("the directory the user came from is saved, not the one previewed: %v", *saved)
	}
	h.RequireNotContains("+ Add current dir")
}

func TestFavoritesRemoveFromTheFile(t *testing.T) {
	dir := tree(t)
	h, saved := openWithFavorites(t, dir, ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: dir})
	h.Press("alt+f")
	h.RequireContains(filepath.Base(dir)).RequireNotContains("+ Add current dir")
	h.Press("down", "down", "delete") // /, ~, then the saved one
	if len(*saved) != 0 {
		t.Fatalf("the favorite is removed from the file: %v", *saved)
	}
	h.RequireContains("+ Add current dir")
	deleteFavorite = func(ftfav.Favorite) error { return errors.New("read-only") }
	h.Press("down", "up", "delete") // a built-in favorite: still removed from the list
	h.RequireContains("Could not remove")
}

func TestFavoritesAddFailureAndOtherKeys(t *testing.T) {
	dir := tree(t)
	h, _ := openWithFavorites(t, dir)
	addFavorite = func(ftfav.Favorite) error { return errors.New("read-only") }
	h.Press("alt+f", "down", "down", "enter")
	h.RequireContains("Could not save the favorite")
	h.Press("enter")
	h.Press("left")
	if h.Model().Zone() != nav.FocusToContent {
		t.Fatal("Left goes to the file list")
	}
}

func TestFavoritesLoadFailureKeepsTheBuiltInOnes(t *testing.T) {
	dir := tree(t)
	isolateState(t)
	getFavorites = func() ([]ftfav.Favorite, error) { return nil, errors.New("no file") }
	h := navtest.New(t, rootPage(Options{Path: dir}), navtest.WithNav(shellOptions()...), navtest.WithSize(140, 30))
	h.Press("alt+f")
	h.RequireContains("/ root")
}

func TestFavoriteText(t *testing.T) {
	cases := []struct {
		item ftfav.Favorite
		want string
	}{
		{ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "/"}, "/ root"},
		{ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "~"}, "~ home"},
		{ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "/work"}, "/work"},
		{ftfav.Favorite{Store: url.URL{Scheme: "https", User: url.UserPassword("u", "p"), Host: "www.example.com", Path: "/files"}}, "HTTPS: example.com/files"},
	}
	for _, c := range cases {
		if got := favoriteText(c.item); got != c.want {
			t.Errorf("%+v: %q, want %q", c.item, got, c.want)
		}
	}
	if favoriteDetail(ftfav.Favorite{Path: "~"}) == "" || favoriteDetail(ftfav.Favorite{Description: "d"}) != "d" {
		t.Fatal("home shows its path, others their description")
	}
}

func TestFavoritesStoreFor(t *testing.T) {
	sess := &session{store: osStore()}
	f := favoritesScreen{sess: sess}
	if s, p := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "/x"}); s != sess.store || p != "/x" {
		t.Fatal("the current store is reused")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "file", Path: "/mnt"}, Path: "/x"}); s == sess.store || s.RootURL().Scheme != "file" {
		t.Fatal("another file store")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "file"}}); s.RootURL().Scheme != "file" {
		t.Fatal("an empty file path is the root")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "https", Host: "h"}}); s.RootURL().Host != "h" {
		t.Fatal("an HTTP store")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "ftp", Host: "h"}}); s.RootURL().Scheme != "ftp" {
		t.Fatal("an FTP store")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "ftps", Host: "h"}}); s != sess.store {
		t.Fatal("a scheme that cannot be opened keeps the current store")
	}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "gopher"}}); s != sess.store {
		t.Fatal("an unknown scheme keeps the current store")
	}
	f.sess = &session{}
	if s, _ := f.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "gopher"}}); s != nil {
		t.Fatal("without a store there is none to keep")
	}
}

func TestFavoritesScreenEdges(t *testing.T) {
	sess := &session{store: fakeStore{root: url.URL{Scheme: "fake"}}}
	f := newFavoritesScreen(sess, newTreeScreen(sess))
	if f.canAddCurrent() {
		t.Fatal("no current directory, nothing to add")
	}
	if _, ok := f.currentFavorite(); ok {
		t.Fatal("no current directory")
	}
	sess.current = files.NewDirContext(sess.store, "/x", nil)
	f = newFavoritesScreen(sess, newTreeScreen(sess))
	if !f.canAddCurrent() {
		t.Fatal("a directory that is not a favorite")
	}
	if _, ok := f.item(); !ok {
		t.Fatal("the first row is a favorite")
	}
	if !f.AtEdge(widgets.Up) || f.AtEdge(widgets.Left) || len(f.ShortHelp()) != 3 || f.Title() != "Favorites" {
		t.Fatal("boundary, help and title")
	}
	f.prevStore = nil
	if f.canAddCurrent() {
		t.Fatal("no store, nothing to add")
	}
	if s, cmd := f.add(); cmd != nil || s == nil {
		t.Fatal("adding needs a current directory")
	}
	if s, cmd := f.remove(); cmd != nil {
		_ = s
	}
	if s, cmd := f.Update(widgets.ItemSelectedMsg{ID: "other"}); cmd != nil || s == nil {
		t.Fatal("other lists are not the favorites")
	}
	if _, cmd := f.Update(widgets.ItemHighlightedMsg{ID: "other"}); cmd != nil {
		t.Fatal("other lists are not the favorites")
	}
	if _, cmd := f.Update(struct{}{}); cmd != nil {
		t.Fatal("unknown messages")
	}
	if f.Init() == nil {
		t.Fatal("the favorites are loaded at start")
	}
	f.list.SetItems(widgets.MenuItem{ID: addItemID, Label: "add"})
	if _, cmd := f.remove(); cmd != nil {
		t.Fatal("the add row is not a favorite")
	}
	if _, cmd := f.highlighted(widgets.ItemHighlightedMsg{ID: favoritesID}); cmd != nil {
		t.Fatal("the add row is not previewed")
	}
}

func TestFavoritesWithoutSchemeAreLocal(t *testing.T) {
	sess := &session{store: osStore()}
	f := newFavoritesScreen(sess, newTreeScreen(sess))
	f.items = append(f.items, ftfav.Favorite{Path: "/tmp"})
	f.setItems()
	if len(f.list.Items()) != 3 {
		t.Fatalf("the built-in favorites and the saved one: %d", len(f.list.Items()))
	}
	other := favoritesScreen{sess: &session{store: fakeStore{root: url.URL{Scheme: "fake"}}}}
	if s, _ := other.storeFor(ftfav.Favorite{Store: url.URL{Scheme: "file"}}); s.RootURL().Scheme != "file" {
		t.Fatal("a local favorite on another store opens the local store")
	}
}
