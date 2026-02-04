package filetug

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftfav"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type recordApp struct {
	queueUpdateDraw func(func())
	focusCalls      []tview.Primitive
	rootCalls       []tview.Primitive
	stopCalled      bool
}

func (a *recordApp) Run() error { return nil }

func (a *recordApp) QueueUpdateDraw(f func()) {
	if a.queueUpdateDraw != nil {
		a.queueUpdateDraw(f)
		return
	}
	if f != nil {
		f()
	}
}

func (a *recordApp) SetFocus(p tview.Primitive) { a.focusCalls = append(a.focusCalls, p) }

func (a *recordApp) SetRoot(root tview.Primitive, _ bool) { a.rootCalls = append(a.rootCalls, root) }

func (a *recordApp) Stop() { a.stopCalled = true }

func (a *recordApp) EnableMouse(_ bool) {}

type stubStore struct {
	root    url.URL
	entries map[string][]os.DirEntry
	readErr error
}

func (s *stubStore) RootURL() url.URL    { return s.root }
func (s *stubStore) RootTitle() string   { return "Stub" }
func (s *stubStore) GetDirReader(_ context.Context, _ string) (files.DirReader, error) {
	return nil, files.ErrNotImplemented
}
func (s *stubStore) ReadDir(_ context.Context, name string) ([]os.DirEntry, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	if s.entries == nil {
		return nil, nil
	}
	return s.entries[name], nil
}
func (s *stubStore) Delete(_ context.Context, _ string) error    { return files.ErrNotImplemented }
func (s *stubStore) CreateDir(_ context.Context, _ string) error { return files.ErrNotImplemented }
func (s *stubStore) CreateFile(_ context.Context, _ string) error {
	return files.ErrNotImplemented
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHelpModalCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	root := nav.Flex

	modal, helpView, button := createHelpModal(nav, root)
	assert.NotNil(t, modal)
	assert.NotNil(t, helpView)
	assert.NotNil(t, button)

	h := helpView.InputHandler()
	event := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	h(event, func(p tview.Primitive) {})
	assert.NotEmpty(t, app.rootCalls)
	assert.NotEmpty(t, app.focusCalls)

	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	h(event, func(p tview.Primitive) {})

	b := button.InputHandler()
	event = tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone)
	b(event, func(p tview.Primitive) {})
	event = tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	b(event, func(p tview.Primitive) {})
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	b(event, func(p tview.Primitive) {})

	showHelpModal(nav)
	assert.NotEmpty(t, app.rootCalls)
}

func TestInitNavigatorWithPersistedStateCoverage(t *testing.T) {
	withTestGlobalLock(t)
	oldGetState := getState
	oldDefaultClient := http.DefaultClient
	defer func() {
		getState = oldGetState
		http.DefaultClient = oldDefaultClient
	}()

	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("http err")
		}),
	}

	app := &recordApp{}
	nav := NewNavigator(app)

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{Store: "http://127.0.0.1:1", CurrentDir: "http://127.0.0.1:1/path"}, nil
	}
	initNavigatorWithPersistedState(nav)

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{Store: "ftp://127.0.0.1:1", CurrentDir: "/"}, nil
	}
	initNavigatorWithPersistedState(nav)

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{Store: "file:", CurrentDir: "", CurrentDirEntry: "file.txt"}, nil
	}
	initNavigatorWithPersistedState(nav)
	assert.Equal(t, "file.txt", nav.files.currentFileName)

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{CurrentDir: "https://127.0.0.1:1/path"}, errors.New("state err")
	}
	initNavigatorWithPersistedState(nav)

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{CurrentDir: "https://%2"}, nil
	}
	initNavigatorWithPersistedState(nav)
}

func TestFavoritesSetStoreCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	panel := &favoritesPanel{nav: nav}

	current := nav.store
	item := ftfav.Favorite{Store: url.URL{Scheme: "file"}, Path: "/tmp"}
	panel.setStore(item)
	assert.NotNil(t, nav.store)

	item = ftfav.Favorite{Store: url.URL{Scheme: "http", Host: "example.com"}, Path: "/"}
	panel.setStore(item)
	assert.NotNil(t, nav.store)

	item = ftfav.Favorite{Store: url.URL{Scheme: "ftp", Host: "127.0.0.1:1"}, Path: "/"}
	panel.setStore(item)
	assert.NotNil(t, nav.store)

	panel.nav.store = current
	item = ftfav.Favorite{Store: current.RootURL(), Path: "/"}
	panel.setStore(item)
	assert.Equal(t, current, nav.store)

	var nilPanel *favoritesPanel
	assert.Equal(t, "", nilPanel.setStore(ftfav.Favorite{}))

	emptyNav := &Navigator{}
	panel = &favoritesPanel{nav: emptyNav}
	assert.Equal(t, "", panel.setStore(ftfav.Favorite{}))
}

func TestSelectionChangedNavFuncCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	fp := nav.files
	fp.selectionChangedNavFunc(10, 0)

	nav.store = &stubStore{root: url.URL{Scheme: "file", Path: "/"}, entries: map[string][]os.DirEntry{
		"/tmp": {files.NewDirEntry("dir", true)},
	}}
	dirContext := files.NewDirContext(nav.store, "/tmp", []os.DirEntry{files.NewDirEntry("dir", true)})
	fp.rows = NewFileRows(dirContext)
	fp.table = tview.NewTable()
	fp.table.SetContent(fp.rows)
	fp.selectionChangedNavFunc(1, 0)
}

func TestShowDirSummaryCoverage(t *testing.T) {
	fp := &filesPanel{}
	fp.showDirSummary(files.NewEntryWithDirPath(files.NewDirEntry("x", true), "/"))

	app := &recordApp{}
	nav := NewNavigator(app)
	fp = nav.files
	nav.store = nil
	nav.previewer = newPreviewerPanel(nav)
	fp.showDirSummary(files.NewEntryWithDirPath(files.NewDirEntry("x", true), "/"))

	stub := &stubStore{root: url.URL{Scheme: "file", Path: "/"}, readErr: errors.New("read err")}
	nav.store = stub
	fp.showDirSummary(files.NewEntryWithDirPath(files.NewDirEntry("x", true), "/"))

	stub.readErr = nil
	stub.entries = map[string][]os.DirEntry{
		"/": {files.NewDirEntry("b", false), files.NewDirEntry("a", false)},
	}
	fp.showDirSummary(files.NewEntryWithDirPath(files.NewDirEntry("x", true), "/"))
}

func TestSetFocusToContainerCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	nav.SetFocusToContainer(nav.left.index)
	nav.SetFocusToContainer(nav.right.index)
	nav.SetFocusToContainer(1)
	nav.SetFocusToContainer(42)

	navNil := NewNavigator(app)
	navNil.app = nil
	navNil.SetFocusToContainer(1)

	nav.showError(errors.New("boom"))
}

func TestGlobalNavInputCaptureCoverage(t *testing.T) {
	nav := &Navigator{}
	event := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	assert.Equal(t, event, nav.globalNavInputCapture(event))

	app := &recordApp{}
	nav = NewNavigator(app)
	nav.store = &stubStore{root: url.URL{Scheme: "file", Path: "/"}}
	event = tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	assert.Nil(t, nav.globalNavInputCapture(event))
	event = tcell.NewEventKey(tcell.KeyRune, '`', tcell.ModNone)
	assert.Nil(t, nav.globalNavInputCapture(event))
	event = tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	assert.Equal(t, event, nav.globalNavInputCapture(event))
}

func TestGetDirDataCoverage(t *testing.T) {
	nav := NewNavigator(&recordApp{})
	nav.store = nil
	_, err := nav.getDirData(context.Background(), "/tmp")
	assert.Error(t, err)

	stub := &stubStore{root: url.URL{Scheme: "file", Path: "/"}, readErr: errors.New("boom")}
	nav.store = stub
	_, err = nav.getDirData(context.Background(), "/tmp")
	assert.Error(t, err)

	stub.readErr = nil
	stub.entries = map[string][]os.DirEntry{
		"/tmp": {files.NewDirEntry("b", false), files.NewDirEntry("a", true)},
	}
	ctx := context.Background()
	ctxDir, err := nav.getDirData(ctx, "/tmp")
	assert.NoError(t, err)
	assert.Len(t, ctxDir.Children(), 2)
}

func TestInputCaptureCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	nav.store = &stubStore{root: url.URL{Scheme: "file", Path: "/"}}
	nav.files.rows = &FileRows{}

	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyF7, 0, tcell.ModNone)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyF8, 0, tcell.ModNone)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyF10, 0, tcell.ModNone)))

	alt := tcell.ModAlt
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, 'f', alt)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, 'm', alt)))
	event0 := tcell.NewEventKey(tcell.KeyRune, '0', alt)
	assert.Same(t, event0, nav.inputCapture(event0))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, '+', alt)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, '-', alt)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, '/', alt)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, '~', alt)))
	assert.Nil(t, nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, 'x', alt)))
	eventZ := tcell.NewEventKey(tcell.KeyRune, 'z', alt)
	assert.Same(t, eventZ, nav.inputCapture(eventZ))

	event := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	assert.Equal(t, event, nav.inputCapture(event))

	navNil := NewNavigator(app)
	navNil.app = nil
	assert.Equal(t, event, navNil.inputCapture(event))
}

func TestResizeCoverage(t *testing.T) {
	nav := NewNavigator(&recordApp{})
	copy(nav.proportions, defaultProportions)
	nav.activeCol = 0
	nav.resize(increase)
	copy(nav.proportions, defaultProportions)
	nav.activeCol = 1
	nav.resize(increase)
	copy(nav.proportions, defaultProportions)
	nav.activeCol = 2
	nav.resize(increase)
	nav.activeCol = 99
	nav.resize(increase)
}

func TestShowNodeErrorCoverage(t *testing.T) {
	nav := NewNavigator(&recordApp{})
	nav.showNodeError(nil, errors.New("err"))

	node := tview.NewTreeNode("node")
	nav.showNodeError(node, errors.New("boom"))
	assert.True(t, strings.Contains(node.GetText(), "boom"))
}

func TestNewNavigatorPanicOnNilApp(t *testing.T) {
	assert.Panics(t, func() {
		NewNavigator(nil)
	})
}

func TestSetBreadcrumbsCoverage(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	nav.breadcrumbs = nil
	nav.setBreadcrumbs()

	nav.breadcrumbs = crumbs.NewBreadcrumbs(crumbs.NewBreadcrumb("root", func() error { return nil }))
	nav.setBreadcrumbs()
	nav.store = nil
	nav.setBreadcrumbs()

	store := &stubStore{root: url.URL{Scheme: "file", Path: ""}}
	nav.store = store
	nav.breadcrumbs = nil
	nav.breadcrumbs = crumbs.NewBreadcrumbs(crumbs.NewBreadcrumb("root", func() error { return nil }))
	nav.current.SetDir(nil)
	nav.setBreadcrumbs()

	nav.current.SetDir(files.NewDirContext(nav.store, "/", nil))
	nav.setBreadcrumbs()

	nav.current.SetDir(files.NewDirContext(nav.store, "/root/dir//sub", nil))
	nav.setBreadcrumbs()
}

func TestSetFocusToContainerNilAppCoverage(t *testing.T) {
	nav := NewNavigator(&recordApp{})
	nav.app = nil
	nav.SetFocusToContainer(nav.left.index)
}

func TestShowDirSummarySymlinkBranch(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	nav.store = osfile.NewStore("/")
	fp := nav.files
	fp.rows = NewFileRows(files.NewDirContext(nav.store, "/", nil))
	entry := files.NewEntryWithDirPath(files.NewDirEntry("link", false), "/")
	fp.showDirSummary(entry)
}

func TestSelectionChangedNavFuncNilRows(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	nav.files.rows = nil
	nav.files.selectionChangedNavFunc(0, 0)
}

func TestGlobalNavInputCaptureNilApp(t *testing.T) {
	var nav Navigator
	event := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	assert.Equal(t, event, nav.globalNavInputCapture(event))
}

func TestSetStoreFileEmptyRootPath(t *testing.T) {
	app := &recordApp{}
	nav := NewNavigator(app)
	panel := &favoritesPanel{nav: nav}
	item := ftfav.Favorite{Store: url.URL{Scheme: "file", Path: ""}, Path: "/tmp"}
	panel.setStore(item)
}
