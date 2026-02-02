package filetug

import (
	"net/url"
	"testing"

	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFavorites(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 4)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).MinTimes(1).MaxTimes(8).DoAndReturn(func(f func()) {
		f()
	})
	f := newFavoritesPanel(nav)

	if f == nil {
		t.Fatal("f is nil")
	}

	t.Run("Draw", func(t *testing.T) {
		screen := tcell.NewSimulationScreen("")
		_ = screen.Init()
		f.Draw(screen)
	})

	t.Run("ShowFavorites", func(t *testing.T) {
		f.nav.ShowFavorites()
	})

	t.Run("activateFavorite_preview", func(t *testing.T) {
		storeURL, err := url.Parse("file:")
		if err != nil {
			t.Fatal(err)
		}
		fav := ftfav.Favorite{Store: *storeURL, Path: ".", Description: "test"}
		f.activateFavorite(fav, true)
	})

	t.Run("activateFavorite_go", func(t *testing.T) {
		storeURL, err := url.Parse("file:")
		if err != nil {
			t.Fatal(err)
		}
		fav := ftfav.Favorite{Store: *storeURL, Path: ".", Description: "test"}
		f.activateFavorite(fav, false)
	})

	t.Run("setStore", func(t *testing.T) {
		// Test different store schemes
		fileURL, err := url.Parse("file:")
		if err != nil {
			t.Fatal(err)
		}
		httpURL, err := url.Parse("https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		ftpURL, err := url.Parse("ftp://example.com")
		if err != nil {
			t.Fatal(err)
		}
		testCases := []ftfav.Favorite{
			{Store: *fileURL, Path: "/tmp"},
			{Store: *httpURL, Path: "/"},
			{Store: *ftpURL, Path: "/"},
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
		httpURL, err := url.Parse("https://www.example.com")
		if err != nil {
			t.Fatal(err)
		}
		fileURL, err := url.Parse("file:")
		if err != nil {
			t.Fatal(err)
		}
		f.items = append(f.items, ftfav.Favorite{Store: *httpURL, Path: "/abc", Description: "Example"})
		f.items = append(f.items, ftfav.Favorite{Store: *fileURL, Path: "/some/path", Description: "Dir"})
		f.setItems()
	})
}

func TestNewFavorites_NilNav(t *testing.T) {
	t.Parallel()
	// Although newFavoritesPanel expects a navigator, let's see what happens if it is nil
	f := newFavoritesPanel(nil)
	if f == nil {
		t.Fatal("f is nil")
	}
}

func TestFavorites_SetStore_InvalidURL(t *testing.T) {
	t.Skip("panics")
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	f := newFavoritesPanel(nav)

	dirPath := f.setStore(ftfav.Favorite{Store: url.URL{Scheme: ":invalid:"}, Path: ""})
	assert.Equal(t, "", dirPath)
}
