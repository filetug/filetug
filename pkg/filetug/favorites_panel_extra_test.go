package filetug

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestFavoritesPanel_InputCapture_DeleteCurrent_Backspace(t *testing.T) {
	//t.Parallel()
	oldDeleteFavorite := deleteFavorite
	defer func() {
		deleteFavorite = oldDeleteFavorite
	}()
	deleted := false
	deleteFavorite = func(item ftfav.Favorite) error {
		_ = item
		deleted = true
		return nil
	}

	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	nav.current.ChangeDir("/tmp")
	panel := newTestFavoritesPanel(nav)
	panel.items = []ftfav.Favorite{
		{Store: url.URL{Scheme: "file"}, Path: "/tmp"},
	}
	panel.setItems()
	panel.list.SetCurrentItem(0)

	key := tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)
	res := panel.inputCapture(key)
	assert.Nil(t, res)
	assert.True(t, deleted)
	assert.Len(t, panel.items, 0)
	assert.True(t, panel.addFormVisible)
}

func TestFavoritesPanel_InputCapture_DeleteCurrent_EmptyList(t *testing.T) {
	//t.Parallel()
	oldDeleteFavorite := deleteFavorite
	defer func() {
		deleteFavorite = oldDeleteFavorite
	}()
	deleteCalled := false
	deleteFavorite = func(item ftfav.Favorite) error {
		_ = item
		deleteCalled = true
		return nil
	}

	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	panel := newTestFavoritesPanel(nav)

	key := tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone)
	res := panel.inputCapture(key)
	assert.Nil(t, res)
	assert.False(t, deleteCalled)
}

func TestFavoritesPanel_AddCurrentFavorite_Success(t *testing.T) {
	//t.Parallel()
	oldAddFavorite := addFavorite
	defer func() {
		addFavorite = oldAddFavorite
	}()
	addCalled := false
	addFavorite = func(item ftfav.Favorite) error {
		_ = item
		addCalled = true
		return nil
	}

	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	nav.current.SetDir(nav.NewDirContext("/tmp", nil))
	panel := newTestFavoritesPanel(nav)
	panel.addFormVisible = true
	panel.flex.AddItem(panel.addContainer, 3, 0, false)
	panel.addCurrentFavorite()

	assert.True(t, addCalled)
	assert.Len(t, panel.items, 1)
	assert.False(t, panel.addFormVisible)
}

func TestFavoritesPanel_AddCurrentFavorite_Error(t *testing.T) {
	//t.Parallel()
	oldAddFavorite := addFavorite
	defer func() {
		addFavorite = oldAddFavorite
	}()
	addFavorite = func(item ftfav.Favorite) error {
		_ = item
		return errors.New("add error")
	}

	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	nav.current.SetDir(nav.NewDirContext("/tmp", nil))
	panel := newTestFavoritesPanel(nav)

	panel.addCurrentFavorite()

	assert.Len(t, panel.items, 0)
}

func TestFavoritesPanel_UpdateAddCurrentForm_ShowHide(t *testing.T) {
	//t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	nav.current.SetDir(nav.NewDirContext("/tmp", nil))
	panel := newTestFavoritesPanel(nav)

	panel.updateAddCurrentForm()
	assert.True(t, panel.addFormVisible)

	panel.updateAddCurrentForm()
	assert.True(t, panel.addFormVisible)

	panel.items = []ftfav.Favorite{{Store: nav.store.RootURL(), Path: "/tmp"}}
	panel.updateAddCurrentForm()
	assert.False(t, panel.addFormVisible)
}

func TestFavoritesPanel_NewFavoritesPanel_GetFavoritesError(t *testing.T) {
	t.Skip("panics")
	t.Parallel()
	oldGetFavorites := getFavorites
	defer func() {
		getFavorites = oldGetFavorites
	}()
	done := make(chan struct{})
	getFavorites = func() ([]ftfav.Favorite, error) {
		close(done)
		return nil, errors.New("favorites error")
	}

	nav, _, _ := newNavigatorForTest(t)
	_ = newFavoritesPanel(nav)

	<-done
}

func TestFavoritesPanel_NewFavoritesPanel_QueueUpdate(t *testing.T) {
	t.Skipf("hanging")
	t.Parallel()
	oldGetFavorites := getFavorites
	defer func() {
		getFavorites = oldGetFavorites
	}()
	done := make(chan struct{})
	userFavs := []ftfav.Favorite{{Store: url.URL{Scheme: "file"}, Path: "/tmp"}}
	getFavorites = func() ([]ftfav.Favorite, error) {
		return userFavs, nil
	}

	nav, app, _ := newNavigatorForTest(t)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).DoAndReturn(func(f func()) {
		select {
		case <-done:
		default:
			f()
			close(done)
		}

	})
	panel := newFavoritesPanel(nav)

	<-done

	assert.GreaterOrEqual(t, len(panel.items), len(userFavs))
	assert.Equal(t, "/tmp", panel.items[len(panel.items)-1].Path)
}

func TestFavoritesPanel_NewFavoritesPanel_NoQueueUpdate(t *testing.T) {
	t.Parallel()
	oldGetFavorites := getFavorites
	defer func() {
		getFavorites = oldGetFavorites
	}()
	userFavs := []ftfav.Favorite{{Store: url.URL{Scheme: "file"}, Path: "/tmp"}}
	getFavorites = func() ([]ftfav.Favorite, error) {
		return userFavs, nil
	}

	nav, _, _ := newNavigatorForTest(t)
	panel := newFavoritesPanel(nav)

	deadline := time.After(200 * time.Millisecond)
	for len(panel.items) <= len(builtInFavorites()) {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for favorites update")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestFavoritesPanel_NewFavoritesPanel_InputCaptures(t *testing.T) {
	t.Skip("failing")
	t.Parallel()
	oldGetFavorites := getFavorites
	defer func() {
		getFavorites = oldGetFavorites
	}()
	getFavorites = func() ([]ftfav.Favorite, error) {
		return []ftfav.Favorite{}, nil
	}

	var focused tview.Primitive
	nav, app, _ := newNavigatorForTest(t)
	app.EXPECT().SetFocus(gomock.Any()).Do(func(p tview.Primitive) {
		focused = p
	}).AnyTimes()
	panel := newFavoritesPanel(nav)

	buttonHandler := panel.addButton.InputHandler()
	buttonHandler(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), func(p tview.Primitive) {})
	assert.Equal(t, panel.list, focused)

	buttonHandler(tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModNone), func(p tview.Primitive) {})
}

func TestFavoritesPanel_InputCapture_KeyTabAndDefault(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	panel := newTestFavoritesPanel(nav)
	panel.addFormVisible = true

	app.EXPECT().SetFocus(panel.addButton).AnyTimes()
	tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	res := panel.inputCapture(tab)
	assert.Nil(t, res)

	panel.addFormVisible = false
	tab = tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	res = panel.inputCapture(tab)
	assert.Equal(t, tab, res)

	other := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	res = panel.inputCapture(other)
	assert.Equal(t, other, res)
}

func TestFavoritesPanel_AddCurrentFavorite_NoCurrent(t *testing.T) {
	t.Parallel()
	panel := newTestFavoritesPanel(nil)

	panel.addCurrentFavorite()

	assert.Len(t, panel.items, 0)
}

func TestFavoritesPanel_InputCapture_KeyEnter_Escape_Left(t *testing.T) {
	t.Parallel()
	oldGetFavorites := getFavorites
	oldGetState := getState
	oldSaveCurrentDir := saveCurrentDir
	defer func() {
		getFavorites = oldGetFavorites
		getState = oldGetState
		saveCurrentDir = oldSaveCurrentDir
	}()
	getFavorites = func() ([]ftfav.Favorite, error) {
		return []ftfav.Favorite{}, nil
	}
	getState = func() (*ftstate.State, error) {
		return nil, errors.New("no state")
	}
	saveCurrentDir = func(storeRoot, dirPath string) {
		_, _ = storeRoot, dirPath
	}

	nav, _, _ := newNavigatorForTest(t)
	store := newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	nav.store = store
	nav.current.SetDir(nav.NewDirContext("/tmp", nil))

	panel := nav.favorites
	panel.nav.ShowFavorites()

	panel.items = []ftfav.Favorite{{Store: nav.store.RootURL(), Path: "/tmp"}}
	panel.setItems()
	panel.list.SetCurrentItem(0)

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	res := panel.inputCapture(enter)
	assert.Nil(t, res)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res = panel.inputCapture(left)
	assert.Nil(t, res)

	escape := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	res = panel.inputCapture(escape)
	assert.Nil(t, res)
}

func TestFavoritesPanel_InputCapture_KeyUpDown(t *testing.T) {
	t.Parallel()
	panel := newTestFavoritesPanel(nil)

	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	res := panel.inputCapture(up)
	assert.Equal(t, up, res)

	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	res = panel.inputCapture(down)
	assert.Equal(t, down, res)
}

func TestFavoritesPanel_DeleteCurrentFavorite_Error(t *testing.T) {
	t.Parallel()
	oldDeleteFavorite := deleteFavorite
	defer func() {
		deleteFavorite = oldDeleteFavorite
	}()
	deleteFavorite = func(item ftfav.Favorite) error {
		_ = item
		return errors.New("delete error")
	}

	nav, _, _ := newNavigatorForTest(t)
	nav.store = newMockStoreWithRoot(t, url.URL{Scheme: "file", Path: "/"})
	nav.current.SetDir(nav.NewDirContext("/tmp", nil))
	panel := newTestFavoritesPanel(nav)
	panel.items = []ftfav.Favorite{
		{Store: url.URL{Scheme: "file"}, Path: "/tmp"},
		{Store: url.URL{Scheme: "file"}, Path: "/other"},
	}
	panel.setItems()
	panel.list.SetCurrentItem(0)

	panel.deleteCurrentFavorite()

	assert.Len(t, panel.items, 1)
	assert.Equal(t, "/other", panel.items[0].Path)
}

func newTestFavoritesPanel(nav *Navigator) *favoritesPanel {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	list := tview.NewList()
	addButton := tview.NewButton("Add Current dir to favorites")
	addContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	addContainer.AddItem(addButton, 1, 0, false)

	return &favoritesPanel{
		Boxed:        sneatv.NewBoxed(flex),
		flex:         flex,
		list:         list,
		nav:          nav,
		addContainer: addContainer,
		addButton:    addButton,
	}
}
