package filetug

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestFavorites(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	f := newFavorites(nav)

	if f == nil {
		t.Fatal("f is nil")
	}

	t.Run("Draw", func(t *testing.T) {
		screen := tcell.NewSimulationScreen("")
		_ = screen.Init()
		f.Draw(screen)
	})

	t.Run("ShowFavorites", func(t *testing.T) {
		f.ShowFavorites()
	})

	t.Run("activateFavorite_preview", func(t *testing.T) {
		fav := favorite{Store: "file:", Path: ".", Description: "test"}
		f.activateFavorite(fav, true)
	})

	t.Run("activateFavorite_go", func(t *testing.T) {
		fav := favorite{Store: "file:", Path: ".", Description: "test"}
		f.activateFavorite(fav, false)
	})

	t.Run("setStore", func(t *testing.T) {
		// Test different store schemes
		testCases := []favorite{
			{Store: "file:", Path: "/tmp"},
			{Store: "https://example.com", Path: "/"},
			{Store: "ftp://example.com", Path: "/"},
		}

		for _, tc := range testCases {
			f.setStore(tc)
		}
	})

	t.Run("inputCapture", func(t *testing.T) {
		// Test Escape
		eventEsc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
		f.inputCapture(eventEsc)

		// Test Enter
		eventEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		f.inputCapture(eventEnter)

		// Test Left
		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		f.inputCapture(eventLeft)

		// Test Up/Down
		eventUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		f.inputCapture(eventUp)
	})

	t.Run("changed", func(t *testing.T) {
		f.changed(0, "", "", 0)
	})

	t.Run("selected", func(t *testing.T) {
		f.selected(f.items[0])
	})

	t.Run("setItems_coverage", func(t *testing.T) {
		f.items = append(f.items, favorite{Store: "https://www.example.com", Path: "/abc", Description: "Example"})
		f.items = append(f.items, favorite{Store: "file:", Path: "/some/path", Description: "Dir"})
		f.setItems()
	})
}

func TestNewFavorites_NilNav(t *testing.T) {
	// Although newFavorites expects a navigator, let's see what happens if it is nil
	// Some methods might panic if nav is nil, but newFavorites itself might not.
	defer func() {
		_ = recover()
	}()
	f := newFavorites(nil)
	if f == nil {
		t.Fatal("f is nil")
	}
}

func TestFavorites_SetStore_InvalidURL(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	f := newFavorites(nav)

	dirPath := f.setStore(favorite{Store: ":invalid:", Path: ""})
	assert.Equal(t, "", dirPath)
}
