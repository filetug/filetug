package filetug

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/filetug/filetug/pkg/viewers"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type mockStoreWithHooks struct {
	root          url.URL
	rootTitle     string
	readDirErr    error
	createDirErr  error
	createFileErr error
	deleteErr     error
	createdDirs   []string
	createdFiles  []string
}

func (m *mockStoreWithHooks) RootTitle() string {
	if m.rootTitle != "" {
		return m.rootTitle
	}
	return "Mock"
}

func newTestDirSummary(nav *Navigator) *viewers.DirSummary {
	filterSetter := viewers.WithDirSummaryFilterSetter(func(filter ftui.Filter) {
		if nav.files == nil {
			return
		}
		nav.files.SetFilter(filter)
	})
	focusLeft := viewers.WithDirSummaryFocusLeft(func() {})
	queueUpdate := viewers.WithDirSummaryQueueUpdateDraw(nav.queueUpdateDraw)
	colorByExt := viewers.WithDirSummaryColorByExt(GetColorByFileExt)
	return viewers.NewDirSummary(nav.app, filterSetter, focusLeft, queueUpdate, colorByExt)
}
func (m *mockStoreWithHooks) RootURL() url.URL { return m.root }

func (m *mockStoreWithHooks) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	_, _ = ctx, name
	if m.readDirErr != nil {
		return nil, m.readDirErr
	}
	return nil, nil
}

func (m *mockStoreWithHooks) CreateDir(ctx context.Context, p string) error {
	_, _ = ctx, p
	m.createdDirs = append(m.createdDirs, p)
	return m.createDirErr
}

func (m *mockStoreWithHooks) CreateFile(ctx context.Context, p string) error {
	_, _ = ctx, p
	m.createdFiles = append(m.createdFiles, p)
	return m.createFileErr
}

func (m *mockStoreWithHooks) Delete(ctx context.Context, p string) error {
	_, _ = ctx, p
	return m.deleteErr
}

type mockDirEntryInfo struct {
	name  string
	isDir bool
	info  os.FileInfo
	err   error
}

func (m mockDirEntryInfo) Name() string      { return m.name }
func (m mockDirEntryInfo) IsDir() bool       { return m.isDir }
func (m mockDirEntryInfo) Type() os.FileMode { return 0 }
func (m mockDirEntryInfo) Info() (os.FileInfo, error) {
	return m.info, m.err
}

type nilFileInfo struct{}

func (n *nilFileInfo) Name() string       { return "nil" }
func (n *nilFileInfo) Size() int64        { return 0 }
func (n *nilFileInfo) Mode() os.FileMode  { return 0 }
func (n *nilFileInfo) ModTime() time.Time { return time.Time{} }
func (n *nilFileInfo) IsDir() bool        { return false }
func (n *nilFileInfo) Sys() interface{}   { return nil }

type failingStore struct {
	root      url.URL
	failAfter int
	calls     int
	failOn    string
}

func (f *failingStore) RootTitle() string { return "Failing" }
func (f *failingStore) RootURL() url.URL  { return f.root }

func (f *failingStore) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	_, _ = ctx, name
	return nil, nil
}

func (f *failingStore) CreateDir(ctx context.Context, p string) error {
	_, _ = ctx, p
	if f.failOn != "" && strings.Contains(p, f.failOn) {
		return errors.New("fail")
	}
	f.calls++
	if f.failAfter > 0 && f.calls >= f.failAfter {
		return errors.New("fail")
	}
	return nil
}

func (f *failingStore) CreateFile(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}

func (f *failingStore) Delete(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}

func TestBottomGetAltMenuItemsExitAction(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	stopCalled := false
	nav.stopApp = func() {
		stopCalled = true
	}

	oldExit := exitApp
	exitCode := -1
	exitApp = func(code int) {
		exitCode = code
	}
	defer func() {
		exitApp = oldExit
	}()

	items := nav.bottom.getAltMenuItems()
	var exitAction func()
	for _, item := range items {
		if item.Title == "Exit" {
			exitAction = item.Action
			break
		}
	}
	if exitAction == nil {
		t.Fatal("expected exit action")
	}

	exitAction()
	assert.True(t, stopCalled)
	assert.Equal(t, 0, exitCode)
}

func TestDirSummary_UpdateTableAndGetSizes_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	emptyExt := &viewers.ExtStat{
		ID: "",
		GroupStats: viewers.GroupStats{
			Count:     1,
			TotalSize: 10,
		},
	}
	multiExt1 := &viewers.ExtStat{
		ID: ".a",
		GroupStats: viewers.GroupStats{
			Count:     1,
			TotalSize: 20,
		},
	}
	multiExt2 := &viewers.ExtStat{
		ID: ".b",
		GroupStats: viewers.GroupStats{
			Count:     2,
			TotalSize: 30,
		},
	}
	groupSingle := &viewers.ExtensionsGroup{
		ID:         "Single",
		Title:      "Singles",
		GroupStats: &viewers.GroupStats{Count: 1, TotalSize: 10},
		ExtStats:   []*viewers.ExtStat{emptyExt},
	}
	groupMulti := &viewers.ExtensionsGroup{
		ID:         "Multi",
		Title:      "Multis",
		GroupStats: &viewers.GroupStats{Count: 1, TotalSize: 50},
		ExtStats:   []*viewers.ExtStat{multiExt1, multiExt2},
	}
	ds.ExtGroups = []*viewers.ExtensionsGroup{groupSingle, groupMulti}

	ds.UpdateTable()

	cell := ds.ExtTable.GetCell(1, 1)
	assert.Contains(t, cell.Text, "<no extension>")

	nilInfoEntry := mockDirEntryInfo{name: "nil.txt", info: nil}
	var typedNil *nilFileInfo
	typedNilEntry := mockDirEntryInfo{name: "typednil.txt", info: typedNil}
	okInfo := mockDirEntryInfo{name: "size.txt", info: mockFileInfo{size: 5}}
	entries := []os.DirEntry{nilInfoEntry, typedNilEntry, okInfo}
	ds.SetDir("/test", entries)
	err := ds.GetSizes()
	assert.NoError(t, err)
}

func TestDirSummary_InputCapture_MoreCoverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
		mockDirEntry{name: "b.log", isDir: false},
		mockDirEntry{name: "c.png", isDir: false},
		mockDirEntry{name: "d.jpg", isDir: false},
	}
	ds.SetDir("/test", entries)

	eventDown := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	eventUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)

	ds.ExtTable.Select(2, 0)
	res := ds.InputCapture(eventDown)
	assert.Nil(t, res)

	ds.ExtTable.Select(1, 0)
	_ = ds.InputCapture(eventUp)
}

func TestFavorites_SetItems_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	f := newFavorites(nav)

	f.items = []favorite{
		{Store: "", Path: "/", Description: "root"},
		{Store: "file:", Path: "~", Description: "home"},
		{Store: "file:", Path: "/tmp", Description: "tmp"},
		{Store: "https://www.example.com", Path: "/docs", Description: "docs"},
	}
	f.setItems()

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	handler := f.list.InputHandler()
	f.list.SetCurrentItem(0)
	handler(enter, func(p tview.Primitive) {})

	itemsField := reflect.ValueOf(f.list).Elem().FieldByName("items")
	itemsValue := reflect.NewAt(itemsField.Type(), unsafe.Pointer(itemsField.UnsafeAddr())).Elem()
	if itemsValue.Len() > 0 {
		itemValue := itemsValue.Index(0)
		selectedField := itemValue.Elem().FieldByName("Selected")
		selectedValue := reflect.NewAt(selectedField.Type(), unsafe.Pointer(selectedField.UnsafeAddr())).Elem()
		selectedFunc := selectedValue.Interface().(func())
		selectedFunc()
	}

	event := tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone)
	res := f.inputCapture(event)
	assert.Equal(t, event, res)
}

func TestFilesPanel_GetCurrentEntry_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	rows := &FileRows{
		VisibleEntries: []files.EntryWithDirPath{
			{DirEntry: mockDirEntry{name: "file.txt", isDir: false}},
		},
	}
	fp.rows = rows
	fp.table.Select(2, 0)
	entry := fp.GetCurrentEntry()
	assert.Nil(t, entry)

	fp.table.Select(0, 0)
	fp.rows.Dir = nil
	entry = fp.GetCurrentEntry()
	assert.Nil(t, entry)
}

func TestFilesPanel_DoLoadingAnimation_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	loading := tview.NewTableCell("")
	fp.table.SetCell(1, 0, loading)

	done := make(chan struct{})
	nav.queueUpdateDraw = func(f func()) {
		f()
		doneCell := tview.NewTableCell("done")
		fp.table.SetCell(1, 0, doneCell)
	}

	go func() {
		fp.doLoadingAnimation(loading)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for loading animation")
	}

	fp.nav = nil
	fp.table.SetCell(1, 0, loading)
	done = make(chan struct{})
	go func() {
		fp.doLoadingAnimation(loading)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	stopCell := tview.NewTableCell("stop")
	fp.table.SetCell(1, 0, stopCell)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for loading animation without queueUpdateDraw")
	}
}

func TestFilesPanel_UpdateGitStatuses_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.gitStatusCache = make(map[string]*gitutils.RepoStatus)
	fp := newFiles(nav)
	ctx := context.Background()

	fp.nav = nil
	fp.updateGitStatuses(ctx, &DirContext{})

	fp.nav = nav
	fp.rows = nil
	fp.updateGitStatuses(ctx, &DirContext{})

	fp.rows = NewFileRows(&DirContext{})
	fp.updateGitStatuses(ctx, nil)

	nav.store = mockStore{root: url.URL{Scheme: "http"}}
	fp.rows = NewFileRows(&DirContext{})
	nonFileDir := &DirContext{Path: t.TempDir()}
	fp.updateGitStatuses(ctx, nonFileDir)

	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}
	noRepoDir := &DirContext{Path: t.TempDir()}
	fp.updateGitStatuses(ctx, noRepoDir)

	badRepoDir := t.TempDir()
	gitDir := filepath.Join(badRepoDir, ".git")
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)
	badRepoContext := &DirContext{Path: badRepoDir}
	fp.updateGitStatuses(ctx, badRepoContext)

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)
	filePath := filepath.Join(repoDir, "file.txt")
	writeErr := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, writeErr)

	dirContext := &DirContext{Path: repoDir}
	rows := NewFileRows(dirContext)
	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("file.txt", false),
		Dir:      repoDir,
	}
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	fp.rows = rows
	fp.table.SetContent(rows)

	drawCalled := make(chan struct{})
	nav.queueUpdateDraw = func(f func()) {
		f()
		select {
		case <-drawCalled:
		default:
			close(drawCalled)
		}
	}
	fp.updateGitStatuses(ctx, dirContext)

	select {
	case <-drawCalled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for git status update")
	}

	nav.queueUpdateDraw = nil
	fp.updateGitStatuses(ctx, dirContext)
	time.Sleep(50 * time.Millisecond)
}

func TestFilesPanel_UpdateGitStatuses_Branches(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.gitStatusCache = make(map[string]*gitutils.RepoStatus)
	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}
	fp := newFiles(nav)

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	filePath := filepath.Join(repoDir, "file.txt")
	writeErr := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, writeErr)

	dirContext := &DirContext{Path: repoDir}
	rows := NewFileRows(dirContext)
	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("file.txt", false),
		Dir:      repoDir,
	}
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	fp.rows = rows
	fp.table.SetContent(rows)

	clearCache := func() {
		nav.gitStatusCacheMu.Lock()
		nav.gitStatusCache = make(map[string]*gitutils.RepoStatus)
		nav.gitStatusCacheMu.Unlock()
	}

	oldGetFileStatus := getFileStatus
	defer func() {
		getFileStatus = oldGetFileStatus
	}()

	getFileStatus = func(ctx context.Context, repo *git.Repository, path string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, path
		return nil
	}
	clearCache()
	fp.updateGitStatuses(context.Background(), dirContext)
	time.Sleep(50 * time.Millisecond)

	status := &gitutils.RepoStatus{
		Branch: "main",
		DirGitChangesStats: gitutils.DirGitChangesStats{
			FilesChanged: 1,
		},
	}
	getFileStatus = func(ctx context.Context, repo *git.Repository, path string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, path
		return status
	}
	fullPath := entry.FullName()
	statusText := nav.gitStatusText(status, fullPath, false)
	updated := rows.SetGitStatusText(fullPath, statusText)
	assert.True(t, updated)
	clearCache()
	fp.updateGitStatuses(context.Background(), dirContext)
	time.Sleep(50 * time.Millisecond)

	nav.queueUpdateDraw = nil
	rows.SetGitStatusText(fullPath, "")
	clearCache()
	fp.updateGitStatuses(context.Background(), dirContext)
	time.Sleep(50 * time.Millisecond)

	rows.gitStatusText = make(map[string]string)
	otherRows := NewFileRows(dirContext)
	done := make(chan struct{})
	nav.queueUpdateDraw = func(f func()) {
		fp.rows = otherRows
		f()
		close(done)
	}
	clearCache()
	fp.updateGitStatuses(context.Background(), dirContext)
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for queueUpdateDraw")
	}
}

func TestFilesPanel_SelectionChanged_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := nav.files

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	dirErr := os.Mkdir(subDir, 0755)
	assert.NoError(t, dirErr)
	filePath := filepath.Join(tempDir, "file.txt")
	writeErr := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, writeErr)

	dirEntry := files.NewDirEntry("sub", true)
	modTime := files.ModTime(time.Now())
	fileEntry := files.NewDirEntry("file.txt", false, files.Size(1), modTime)
	rows := NewFileRows(&DirContext{Path: tempDir})
	rows.AllEntries = []files.EntryWithDirPath{
		{DirEntry: dirEntry, Dir: tempDir},
		{DirEntry: fileEntry, Dir: tempDir},
	}
	rows.VisibleEntries = rows.AllEntries
	fp.rows = rows
	fp.table.SetContent(rows)

	cell := tview.NewTableCell("no-ref")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChanged(1, 0)

	dirCell := tview.NewTableCell("dir")
	dirCell.SetReference(&rows.AllEntries[0])
	fp.table.SetCell(1, 0, dirCell)
	fp.selectionChanged(1, 0)

	fileCell := tview.NewTableCell("file")
	fileCell.SetReference(&rows.AllEntries[1])
	fp.table.SetCell(2, 0, fileCell)
	fp.selectionChanged(2, 0)
}

func TestHelpModal_InputCapture(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	rootCalled := false
	focusCalled := false
	nav.setAppRoot = func(root tview.Primitive, fullscreen bool) {
		_, _ = root, fullscreen
		rootCalled = true
	}
	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
		focusCalled = true
	}

	modal, helpView, button := createHelpModal(nav, nav.Flex)
	assert.NotNil(t, modal)

	esc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	helpHandler := helpView.GetInputCapture()
	helpHandler(esc)
	assert.True(t, rootCalled)
	assert.True(t, focusCalled)

	rootCalled = false
	focusCalled = false
	f1 := tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone)
	helpHandler(f1)
	assert.True(t, rootCalled)
	assert.True(t, focusCalled)

	rootCalled = false
	focusCalled = false
	buttonHandler := button.InputHandler()
	buttonHandler(f1, func(p tview.Primitive) {})
	assert.True(t, rootCalled)
	assert.True(t, focusCalled)

	rootCalled = false
	focusCalled = false
	buttonHandler(esc, func(p tview.Primitive) {})
	assert.True(t, rootCalled)
	assert.True(t, focusCalled)

	otherButton := tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModNone)
	buttonHandler(otherButton, func(p tview.Primitive) {})

	other := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	res := helpHandler(other)
	assert.Equal(t, other, res)
}

func TestCreateLeft_FocusFuncs(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.activeCol = -1

	nav.favorites.flex.Box.Focus(func(p tview.Primitive) {})
	assert.Equal(t, 0, nav.activeCol)

	nav.activeCol = -1
	nav.favoritesFocusFunc()
	assert.Equal(t, 0, nav.activeCol)
}

func TestNavigator_InputCapture_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	f7 := tcell.NewEventKey(tcell.KeyF7, 0, tcell.ModNone)
	res := nav.inputCapture(f7)
	assert.Nil(t, res)

	f10 := tcell.NewEventKey(tcell.KeyF10, 0, tcell.ModNone)
	res = nav.inputCapture(f10)
	assert.Nil(t, res)

	altZero := tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModAlt)
	res = nav.inputCapture(altZero)
	assert.Equal(t, altZero, res)

	altUnknown := tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModAlt)
	res = nav.inputCapture(altUnknown)
	assert.Equal(t, altUnknown, res)
}

func TestNewNavigator_StateError(t *testing.T) {
	oldGetState := getState
	defer func() {
		getState = oldGetState
	}()

	getState = func() (*ftstate.State, error) {
		return &ftstate.State{
			Store:           "file:",
			CurrentDir:      "/tmp",
			CurrentDirEntry: "file.txt",
		}, errors.New("state error")
	}

	app := tview.NewApplication()
	nav := NewNavigator(app)
	assert.NotNil(t, nav)
}

func TestNavigator_InputCapture_MoreKeys(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.setAppRoot = func(root tview.Primitive, fullscreen bool) {
		_, _ = root, fullscreen
	}
	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
	}
	stopCalled := false
	nav.stopApp = func() {
		stopCalled = true
	}
	nav.activeCol = 1
	nav.files.rows = &FileRows{}

	f1 := tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone)
	nav.inputCapture(f1)

	f8 := tcell.NewEventKey(tcell.KeyF8, 0, tcell.ModNone)
	nav.inputCapture(f8)

	altSlash := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModAlt)
	nav.inputCapture(altSlash)

	altHome := tcell.NewEventKey(tcell.KeyRune, '~', tcell.ModAlt)
	nav.inputCapture(altHome)

	altX := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt)
	nav.inputCapture(altX)
	assert.True(t, stopCalled)
}

func TestNavigator_GetCurrentBrowser_DefaultBranch(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.activeCol = 2
	browser := nav.getCurrentBrowser()
	assert.Nil(t, browser)
}

func TestNavigator_GetGitStatus_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	cached := &gitutils.RepoStatus{Branch: "main"}
	fullPath := "/cached/path"
	nav.gitStatusCache[fullPath] = cached
	status := nav.getGitStatus(context.Background(), nil, fullPath, true)
	assert.Equal(t, cached, status)

	noRepoPath := t.TempDir()
	status = nav.getGitStatus(context.Background(), nil, noRepoPath, true)
	assert.Nil(t, status)

	badRepoPath := t.TempDir()
	gitDir := filepath.Join(badRepoPath, ".git")
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)
	status = nav.getGitStatus(context.Background(), nil, badRepoPath, true)
	assert.Nil(t, status)

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	oldGetDirStatus := getDirStatus
	oldGetFileStatus := getFileStatus
	defer func() {
		getDirStatus = oldGetDirStatus
		getFileStatus = oldGetFileStatus
	}()

	getDirStatus = func(ctx context.Context, repo *git.Repository, dir string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, dir
		return &gitutils.RepoStatus{Branch: "main"}
	}
	getFileStatus = func(ctx context.Context, repo *git.Repository, filePath string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, filePath
		return &gitutils.RepoStatus{Branch: "main"}
	}

	status = nav.getGitStatus(context.Background(), repo, repoDir, true)
	assert.NotNil(t, status)

	status = nav.getGitStatus(context.Background(), repo, repoDir, false)
	assert.NotNil(t, status)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelPath := filepath.Join(repoDir, "cancel")
	status = nav.getGitStatus(cancelCtx, repo, cancelPath, true)
	assert.Nil(t, status)
}

func TestNavigator_GetGitStatus_OpenRepo(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	oldGetDirStatus := getDirStatus
	defer func() {
		getDirStatus = oldGetDirStatus
	}()

	getDirStatus = func(ctx context.Context, repo *git.Repository, dir string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, dir
		return &gitutils.RepoStatus{Branch: "main"}
	}

	status := nav.getGitStatus(context.Background(), nil, repoDir, true)
	assert.NotNil(t, status)
}

func TestNavigator_GitStatusText_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	empty := nav.gitStatusText(nil, "/tmp", true)
	assert.Equal(t, "", empty)

	status := &gitutils.RepoStatus{Branch: "main"}
	text := nav.gitStatusText(status, "/tmp/not-a-repo", true)
	assert.Equal(t, "", text)
}

func TestNavigator_SetBreadcrumbs_EmptyPath(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = mockStore{root: url.URL{}}
	nav.current.dir = "/"
	nav.setBreadcrumbs()
}

func TestScriptsPanel_And_NestedDirsGenerator(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	nav.showScriptsPanel()
	panel := nav.right.content
	scripts, ok := panel.(*scriptsPanel)
	assert.True(t, ok)

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	listHandler := scripts.list.InputHandler()
	listHandler(enter, func(p tview.Primitive) {})

	ndgPanel := nav.right.content
	ndg, ok := ndgPanel.(*nestedDirsGeneratorPanel)
	assert.True(t, ok)

	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
	}

	cancelButton := ndg.form.GetButton(1)
	cancelHandler := cancelButton.InputHandler()
	cancelHandler(enter, func(p tview.Primitive) {})
}

func TestGeneratedNestedDirs_Coverage(t *testing.T) {
	store := &mockStoreWithHooks{}
	ctx := context.Background()
	err := GeneratedNestedDirs(ctx, store, "/tmp", "", 0, 0)
	assert.NoError(t, err)

	store.createDirErr = errors.New("fail")
	err = GeneratedNestedDirs(ctx, store, "/tmp", "", 1, 1)
	assert.Error(t, err)
}

func TestNewPanel_InputCapture_Create(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	store := &mockStoreWithHooks{root: url.URL{Scheme: "file", Path: "/"}}
	nav.store = store
	nav.current.dir = "/tmp"

	panel := NewNewPanel(nav)
	var focused tview.Primitive
	nav.setAppFocus = func(p tview.Primitive) {
		focused = p
	}

	panel.Show()
	panel.input.SetText("")
	panel.createDir()
	panel.createFile()

	panel.input.SetText("newdir")
	panel.createDir()
	assert.Len(t, store.createdDirs, 1)

	store.createDirErr = errors.New("fail")
	panel.input.SetText("faildir")
	panel.createDir()

	panel.input.SetText("newfile")
	panel.createFile()
	assert.Len(t, store.createdFiles, 1)

	store.createFileErr = errors.New("fail")
	panel.input.SetText("failfile")
	panel.createFile()

	buttonEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	panel.input.SetText("buttondir")
	dirHandler := panel.createDirBtn.InputHandler()
	dirHandler(buttonEnter, func(p tview.Primitive) {})

	panel.input.SetText("buttonfile")
	fileHandler := panel.createFileBtn.InputHandler()
	fileHandler(buttonEnter, func(p tview.Primitive) {})

	tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	inputCapture := panel.input.GetInputCapture()
	inputCapture(tab)
	assert.Equal(t, panel.createDirBtn, focused)

	panel.createDirBtn.Focus(func(p tview.Primitive) {})
	inputCapture(tab)
	assert.Equal(t, panel.createFileBtn, focused)

	panel.createDirBtn.Blur()
	panel.createFileBtn.Focus(func(p tview.Primitive) {})
	inputCapture(tab)
	assert.Equal(t, panel.input, focused)

	dKey := tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone)
	inputCapture(dKey)

	fKey := tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone)
	inputCapture(fKey)

	altD := tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModAlt)
	res := inputCapture(altD)
	assert.Equal(t, altD, res)

	altF := tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModAlt)
	res = inputCapture(altF)
	assert.Equal(t, altF, res)

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	panel.input.SetText("enterfile")
	inputHandler := panel.input.InputHandler()
	inputHandler(enter, func(p tview.Primitive) {})

	esc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	inputHandler(esc, func(p tview.Primitive) {})
}

func TestTree_InputCapture_SetSearch_GetCurrentEntry_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root")
	child := tview.NewTreeNode("child")
	child.SetReference("/root/child")
	root.AddChild(child)
	tree.tv.SetCurrentNode(child)

	right := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	res := tree.inputCapture(right)
	assert.Nil(t, res)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res = tree.inputCapture(left)
	assert.Nil(t, res)

	nonString := tview.NewTreeNode("non")
	nonString.SetReference(123)
	tree.tv.SetCurrentNode(nonString)
	res = tree.inputCapture(left)
	assert.Equal(t, left, res)

	tree.tv.SetCurrentNode(root)
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	res = tree.inputCapture(enter)
	assert.Nil(t, res)

	tree.tv.SetCurrentNode(nonString)
	res = tree.inputCapture(enter)
	assert.Equal(t, enter, res)

	tree.tv.SetCurrentNode(root)
	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	res = tree.inputCapture(up)
	assert.Nil(t, res)

	tree.searchPattern = "ab"
	back := tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)
	res = tree.inputCapture(back)
	assert.Nil(t, res)

	esc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	res = tree.inputCapture(esc)
	assert.Nil(t, res)

	space := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	res = tree.inputCapture(space)
	assert.Equal(t, space, res)

	key := tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone)
	res = tree.inputCapture(key)
	assert.Nil(t, res)

	tree.SetSearch("child")
	tree.SetSearch("zz")

	tree.tv.SetCurrentNode(nil)
	entry := tree.GetCurrentEntry()
	assert.Nil(t, entry)

	tree.tv.SetCurrentNode(root)
	root.SetReference(nil)
	entry = tree.GetCurrentEntry()
	assert.Nil(t, entry)

	root.SetReference("/root")
	entry = tree.GetCurrentEntry()
	if entry == nil {
		t.Fatal("expected entry to be non-nil after setting reference")
	}
	expectedDir := path.Dir("/root")
	assert.Equal(t, expectedDir, entry.Dir)
}

func TestTree_SetCurrentDir_And_DoLoadingAnimation_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}
	tree.setCurrentDir("/")

	oldHome := userHomeDir
	userHomeDir = "/home/user"
	defer func() {
		userHomeDir = oldHome
	}()

	tree.setCurrentDir("/home/user")
	tree.setCurrentDir("/tmp")

	loading := tview.NewTreeNode(" Loading...")
	tree.rootNode.ClearChildren()
	tree.rootNode.AddChild(loading)
	done := make(chan struct{})
	nav.queueUpdateDraw = func(f func()) {
		f()
		tree.rootNode.ClearChildren()
	}
	go func() {
		tree.doLoadingAnimation(loading)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for tree loading animation")
	}

	nav.queueUpdateDraw = nil
	tree.rootNode.AddChild(loading)
	done = make(chan struct{})
	go func() {
		tree.doLoadingAnimation(loading)
		close(done)
	}()
	time.Sleep(60 * time.Millisecond)
	tree.rootNode.ClearChildren()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for tree loading animation without queueUpdateDraw")
	}

	dirEntry := &treeDirEntry{name: "name", isDir: true}
	dirName := dirEntry.Name()
	isDir := dirEntry.IsDir()
	assert.Equal(t, "name", dirName)
	assert.True(t, isDir)
}

func TestFilesPanel_InputCapture_ExtraBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
	}
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.table.Select(1, 0)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res := fp.inputCapture(left)
	assert.Nil(t, res)

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	res = fp.inputCapture(enter)
	assert.Equal(t, enter, res)

	other := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	res = fp.inputCapture(other)
	assert.Equal(t, other, res)
}

func TestFilesPanel_InputCapture_KeyUp_NoMoveFocus(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
	}
	nav.current.dir = "/tmp"
	fp := newFiles(nav)

	cell := tview.NewTableCell("..")
	fp.table.SetCell(0, 0, cell)
	fp.table.Select(0, 0)

	event := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	res := fp.inputCapture(event)
	assert.Equal(t, event, res)
}

func TestFilesPanel_InputCapture_KeyEnterEntry(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := nav.files

	dirEntry := files.NewDirEntry("tmp", true)
	entry := files.NewEntryWithDirPath(dirEntry, "/")
	cell := tview.NewTableCell("file")
	cell.SetReference(entry)
	fp.table.SetCell(1, 0, cell)
	fp.table.SetSelectionChangedFunc(nil)
	fp.table.Select(1, 0)

	event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	res := fp.inputCapture(event)
	assert.Nil(t, res)
}

func TestFilesPanel_SelectionChangedNavFunc_RefNil(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestShowNestedDirsGenerator_PanelCancel(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	nav.setAppFocus = func(p tview.Primitive) {
		_ = p
	}

	active := nav.files
	showNestedDirsGenerator(nav)
	panel := nav.right.content
	ndg, ok := panel.(*nestedDirsGeneratorPanel)
	assert.True(t, ok)

	cancelButton := ndg.form.GetButton(1)
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	cancelHandler := cancelButton.InputHandler()
	cancelHandler(enter, func(p tview.Primitive) {})

	nav.right.SetContent(active)
}

func TestNavigator_ShowNewPanel(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.showNewPanel()
	assert.NotNil(t, nav.right.content)
}

func TestNavigator_UpdateGitStatus_NodeNil(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.updateGitStatus(context.Background(), nil, "/tmp", nil, "prefix")
}

func TestNavigator_ShowDir_NodeNil(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.queueUpdateDraw = func(f func()) {
		f()
	}
	nav.store = mockStore{root: url.URL{Scheme: "http"}}
	ctx := context.Background()
	nav.showDir(ctx, nil, "/tmp", false)
	time.Sleep(50 * time.Millisecond)
}

func TestPreviewerPanel_SetPreviewer_Switch(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	panel := newPreviewerPanel(nav)

	first := viewers.NewTextPreviewer()
	panel.setPreviewer(first)

	second := viewers.NewJsonPreviewer()
	panel.setPreviewer(second)
	panel.setPreviewer(nil)
}

func TestFilesPanel_SelectionChangedNavFunc_NilRef(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestNavigator_DeleteEntries_Error(t *testing.T) {
	ctx := context.Background()
	store := &mockStoreWithHooks{deleteErr: errors.New("fail")}
	err := deleteEntries(ctx, store, []string{"/tmp/file"}, func(progress OperationProgress) {})
	assert.Error(t, err)
}

func TestNavigator_GitStatusText_HasChanges(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	status := &gitutils.RepoStatus{
		Branch: "main",
		DirGitChangesStats: gitutils.DirGitChangesStats{
			FilesChanged: 1,
		},
	}
	text := nav.gitStatusText(status, "/tmp/not-a-repo", false)
	assert.NotEqual(t, "", text)
}

func TestNavigator_SetBreadcrumbs_PathItems(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = mockStore{root: url.URL{Path: "/root"}}
	nav.current.dir = "/root/dir//child"
	nav.setBreadcrumbs()
}

func TestNavigator_SetBreadcrumbs_TitleTrim(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = &mockStoreWithHooks{
		root:      url.URL{Path: "/root"},
		rootTitle: "Root/",
	}
	nav.current.dir = "/root/child"
	nav.setBreadcrumbs()
}

func TestNavigator_BreadcrumbActions(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	err := nav.breadcrumbs.GoHome()
	assert.NoError(t, err)

	nav.store = mockStore{root: url.URL{Path: "/"}}
	nav.current.dir = "/tmp"
	nav.setBreadcrumbs()
	err = nav.breadcrumbs.GoHome()
	assert.NoError(t, err)

	itemsField := reflect.ValueOf(nav.breadcrumbs).Elem().FieldByName("items")
	itemsValue := reflect.NewAt(itemsField.Type(), unsafe.Pointer(itemsField.UnsafeAddr())).Elem()
	if itemsValue.Len() > 1 {
		itemValue := itemsValue.Index(1)
		item := itemValue.Interface().(crumbs.Breadcrumb)
		err = item.Action()
		assert.NoError(t, err)
	}
}

func TestNavigator_GetGitStatus_ContextCancel(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	oldGetDirStatus := getDirStatus
	defer func() {
		getDirStatus = oldGetDirStatus
	}()

	getDirStatus = func(ctx context.Context, repo *git.Repository, dir string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, dir
		return &gitutils.RepoStatus{Branch: "main"}
	}

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	status := nav.getGitStatus(cancelCtx, repo, repoDir, true)
	assert.Nil(t, status)
}

func TestFilesPanel_SelectionChanged_ErrorPath(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	rows := NewFileRows(&DirContext{Path: "/non-existent"})
	entry := files.EntryWithDirPath{DirEntry: files.NewDirEntry("missing.txt", false), Dir: "/non-existent"}
	rows.VisibleEntries = []files.EntryWithDirPath{entry}
	fp.rows = rows
	fp.table.SetContent(rows)

	fp.selectionChanged(1, 0)
}

func TestFilesPanel_SelectionChangedNavFunc_RefMissing(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestTree_InputCapture_Default(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)
	other := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	res := tree.inputCapture(other)
	assert.Nil(t, res)
}

func TestTree_InputCapture_DefaultKey(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)
	key := tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModNone)
	res := tree.inputCapture(key)
	assert.Equal(t, key, res)
}

func TestTree_InputCapture_KeyUp_NotRoot(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root")
	child := tview.NewTreeNode("child")
	child.SetReference("/root/child")
	root.AddChild(child)
	tree.tv.SetCurrentNode(child)

	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	res := tree.inputCapture(up)
	assert.Equal(t, up, res)
}

func TestGeneratedNestedDirs_Recursive(t *testing.T) {
	store := &mockStoreWithHooks{}
	ctx := context.Background()
	err := GeneratedNestedDirs(ctx, store, "/tmp", "Dir%d", 1, 2)
	assert.NoError(t, err)
}

func TestNavigator_SetBreadcrumbs_RootTitle(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = mockStore{root: url.URL{Path: "/root"}}
	nav.current.dir = "/root"
	nav.setBreadcrumbs()
}

func TestNavigator_ShowScriptsPanel_Selection(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.showScriptsPanel()
	panel := nav.right.content
	scripts, ok := panel.(*scriptsPanel)
	assert.True(t, ok)

	selectFunc := scripts.list.InputHandler()
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	selectFunc(enter, func(p tview.Primitive) {})
}

func TestTree_SetSearch_Recursion(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root")
	child := tview.NewTreeNode("alpha")
	child.SetReference("/root/alpha")
	root.AddChild(child)

	tree.SetSearch("zz")
	assert.Equal(t, "", tree.searchPattern)
}

func TestDirSummary_GetSizes_Error(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	entries := []os.DirEntry{
		mockDirEntryInfo{name: "bad.txt", err: errors.New("fail")},
	}
	ds.SetDir("/test", entries)
	err := ds.GetSizes()
	assert.Error(t, err)
}

func TestFilesPanel_SelectionChangedNavFunc_SetsPreview(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := nav.files

	modTime := files.ModTime(time.Now())
	dirEntry := files.NewDirEntry("file.txt", false, files.Size(1), modTime)
	entry := files.EntryWithDirPath{DirEntry: dirEntry, Dir: "/tmp"}
	cell := tview.NewTableCell("file")
	cell.SetReference(&entry)
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestNavigator_ShowDir_Error(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	nav.queueUpdateDraw = func(f func()) {
		f()
	}

	nav.store = &mockStoreWithHooks{
		root: url.URL{Scheme: "file", Path: "/"},
	}
	nav.current.dir = "/tmp"

	node := tview.NewTreeNode("node")
	node.SetReference("/tmp")

	ctx := context.Background()
	nav.showDir(ctx, node, "/tmp", false)
	time.Sleep(50 * time.Millisecond)
}

func TestNavigator_ShowDir_ReadError(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	nav.queueUpdateDraw = func(f func()) {
		f()
	}
	nav.store = &mockStoreWithHooks{
		root:       url.URL{Scheme: "file", Path: "/"},
		readDirErr: errors.New("read error"),
	}
	nav.current.dir = "/tmp"

	node := tview.NewTreeNode("node")
	node.SetReference("/tmp")

	ctx := context.Background()
	nav.showDir(ctx, node, "/tmp", true)
	time.Sleep(50 * time.Millisecond)
}

func TestFilesPanel_SelectionChangedNavFunc_RefNilReuse(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestDirSummary_InputCapture_Left(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	entries := []os.DirEntry{
		mockDirEntry{name: "image.png", isDir: false},
	}
	ds.SetDir("/test", entries)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res := ds.InputCapture(left)
	assert.Nil(t, res)
}

func TestNewPanel_ShowAndFocus(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	panel := NewNewPanel(nav)
	panel.Show()
	panel.Focus(func(p tview.Primitive) {})
}

func TestNavigator_ShowScriptsPanel_InputCapture(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	nav.showScriptsPanel()
	panel := nav.right.content
	scripts, ok := panel.(*scriptsPanel)
	assert.True(t, ok)

	key := tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone)
	handler := scripts.list.InputHandler()
	handler(key, func(p tview.Primitive) {})
}

func TestFilesPanel_SelectionChanged_NilRef(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChanged(1, 0)
}

func TestDirSummary_UpdateTable_SingleExtGroup(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	ext := &viewers.ExtStat{
		ID: ".txt",
		GroupStats: viewers.GroupStats{
			Count:     1,
			TotalSize: 1,
		},
	}
	group := &viewers.ExtensionsGroup{
		ID:         "Text",
		Title:      "Texts",
		GroupStats: &viewers.GroupStats{Count: 1, TotalSize: 1},
		ExtStats:   []*viewers.ExtStat{ext},
	}
	ds.ExtGroups = []*viewers.ExtensionsGroup{group}
	ds.UpdateTable()
}

func TestTree_GetCurrentEntry_RefNil(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference(nil)
	tree.tv.SetCurrentNode(root)

	entry := tree.GetCurrentEntry()
	assert.Nil(t, entry)
}

func TestTree_InputCapture_LeftWithRoot(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root/child")
	tree.tv.SetCurrentNode(root)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res := tree.inputCapture(left)
	assert.Nil(t, res)
}

func TestNavigator_Delete_NoCurrentEntry(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.activeCol = 1
	nav.files.rows = &FileRows{}
	nav.delete()
}

func TestNavigator_Delete_WithError(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	errStore := &mockStoreWithHooks{
		root:      url.URL{Scheme: "file", Path: "/"},
		deleteErr: errors.New("fail"),
	}
	nav.store = errStore
	nav.activeCol = 1

	dirContext := &DirContext{Path: "/tmp", Store: errStore}
	rows := NewFileRows(dirContext)
	dirEntry := files.NewDirEntry("file.txt", false)
	entry := files.EntryWithDirPath{
		DirEntry: dirEntry,
		Dir:      "/tmp",
	}
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	info := files.NewFileInfo(dirEntry)
	rows.Infos = []os.FileInfo{info}
	nav.files.SetRows(rows, false)
	nav.files.table.Select(0, 0)

	nav.delete()
	time.Sleep(50 * time.Millisecond)
}

func TestNavigator_DeleteEntries_Success(t *testing.T) {
	ctx := context.Background()
	store := &mockStoreWithHooks{}
	err := deleteEntries(ctx, store, []string{"/tmp/file"}, func(progress OperationProgress) {})
	assert.NoError(t, err)
}

func TestNavigator_GitStatusText_IsRepoRoot(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	status := &gitutils.RepoStatus{Branch: "main"}
	repoDir := t.TempDir()
	gitDir := filepath.Join(repoDir, ".git")
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)

	text := nav.gitStatusText(status, repoDir, true)
	assert.NotEqual(t, "", text)
}

func TestDirSummary_GetSizes_NilInfo(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	entries := []os.DirEntry{
		mockDirEntryInfo{name: "nil.txt", info: nil},
	}
	ds.SetDir("/test", entries)
	err := ds.GetSizes()
	assert.NoError(t, err)
}

func TestNavigator_ShowDir_NoNode(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}

	ctx := context.Background()
	nav.showDir(ctx, nil, "/tmp", true)
	time.Sleep(50 * time.Millisecond)
}

func TestFilesPanel_SelectionChangedNavFunc_NilRef_Extra(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestNavigator_ShowScriptsPanel_ListShortcut(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.showScriptsPanel()
	panel := nav.right.content
	scripts, ok := panel.(*scriptsPanel)
	assert.True(t, ok)

	key := tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone)
	handler := scripts.list.InputHandler()
	handler(key, func(p tview.Primitive) {})
}

func TestTree_InputCapture_SpaceWithSearch(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)
	tree.searchPattern = "a"
	space := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	res := tree.inputCapture(space)
	assert.Nil(t, res)
}

func TestNavigator_ShowDir_SetsBreadcrumbs(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.queueUpdateDraw = func(f func()) {
		f()
	}
	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}
	ctx := context.Background()
	node := tview.NewTreeNode("node")
	node.SetReference("/tmp")
	nav.showDir(ctx, node, "/tmp", true)
	time.Sleep(50 * time.Millisecond)
}

func TestFilesPanel_SelectionChanged_WithDirAndFile(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := nav.files

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "dir")
	dirErr := os.Mkdir(subDir, 0755)
	assert.NoError(t, dirErr)
	filePath := filepath.Join(tempDir, "file.txt")
	writeErr := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, writeErr)

	dirEntry := files.NewDirEntry("dir", true)
	modTime := files.ModTime(time.Now())
	fileEntry := files.NewDirEntry("file.txt", false, files.Size(1), modTime)
	rows := NewFileRows(&DirContext{Path: tempDir})
	rows.VisibleEntries = []files.EntryWithDirPath{
		{DirEntry: dirEntry, Dir: tempDir},
		{DirEntry: fileEntry, Dir: tempDir},
	}
	fp.rows = rows
	fp.table.SetContent(rows)

	fp.selectionChanged(1, 0)
	fp.selectionChanged(2, 0)
}

func TestDirSummary_InputCapture_NoGroupRefs(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	cell := tview.NewTableCell("no-ref")
	ds.ExtTable.SetCell(0, 1, cell)
	ds.ExtTable.Select(0, 0)
	event := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	res := ds.InputCapture(event)
	assert.Equal(t, event, res)
}

func TestNavigator_GetGitStatus_CacheStore(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	oldGetDirStatus := getDirStatus
	defer func() {
		getDirStatus = oldGetDirStatus
	}()

	getDirStatus = func(ctx context.Context, repo *git.Repository, dir string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, dir
		return &gitutils.RepoStatus{Branch: "main"}
	}

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	status := nav.getGitStatus(context.Background(), repo, repoDir, true)
	assert.NotNil(t, status)

	cached := nav.getGitStatus(context.Background(), repo, repoDir, true)
	assert.NotNil(t, cached)
}

func TestPreviewerPanel_SetPreviewer_RemoveMeta(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	panel := newPreviewerPanel(nav)

	meta := tview.NewTextView()
	main := tview.NewTextView()
	mock := &mockPreviewer{
		meta: meta,
		main: main,
	}
	panel.setPreviewer(mock)
	panel.setPreviewer(nil)
}

type mockPreviewer struct {
	meta tview.Primitive
	main tview.Primitive
}

func (m *mockPreviewer) Preview(entry files.EntryWithDirPath, _ []byte, _ func(func())) {
	_ = entry
}

func (m *mockPreviewer) Main() tview.Primitive { return m.main }
func (m *mockPreviewer) Meta() tview.Primitive { return m.meta }

func TestNavigator_ShowDir_ErrorNode(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.queueUpdateDraw = func(f func()) {
		f()
	}

	nav.store = &mockStoreWithHooks{
		root: url.URL{Scheme: "file", Path: "/"},
	}
	nav.current.dir = "/tmp"
	node := tview.NewTreeNode("node")
	node.SetReference("/tmp")

	ctx := context.Background()
	nav.showDir(ctx, node, "/tmp", true)
	time.Sleep(50 * time.Millisecond)
}

func TestNavigator_SetBreadcrumbs_EmptyRelativePath(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = mockStore{root: url.URL{Path: "/"}}
	nav.current.dir = "/"
	nav.setBreadcrumbs()
}

func TestTree_SetSearch_FirstPrefixed(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root")
	child := tview.NewTreeNode("alpha")
	child.SetReference("/root/alpha")
	root.AddChild(child)

	tree.SetSearch("al")
	assert.Equal(t, "al", tree.searchPattern)
}

func TestTree_SetSearch_FirstContains(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	root := tree.tv.GetRoot()
	root.SetReference("/root")
	child := tview.NewTreeNode("alpha")
	child.SetReference("/root/alpha")
	root.AddChild(child)

	tree.SetSearch("lp")
	assert.Equal(t, "lp", tree.searchPattern)
}

func TestNavigator_ShowScriptsPanel_ListEnter(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.showScriptsPanel()
	panel := nav.right.content
	scripts, ok := panel.(*scriptsPanel)
	assert.True(t, ok)

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	handler := scripts.list.InputHandler()
	handler(enter, func(p tview.Primitive) {})
}

func TestFilesPanel_SelectionChanged_WithError(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	rows := NewFileRows(&DirContext{Path: "/missing"})
	entry := files.EntryWithDirPath{DirEntry: files.NewDirEntry("missing.txt", false), Dir: "/missing"}
	rows.VisibleEntries = []files.EntryWithDirPath{entry}
	fp.rows = rows
	fp.table.SetContent(rows)

	fp.selectionChanged(1, 0)
}

func TestDirSummary_InputCapture_UpAtTop(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	entries := []os.DirEntry{
		mockDirEntry{name: "image.png", isDir: false},
	}
	ds.SetDir("/test", entries)

	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	ds.ExtTable.Select(0, 0)
	res := ds.InputCapture(up)
	assert.Equal(t, up, res)
}

func TestDirSummary_InputCapture_DownAtBottom(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	entries := []os.DirEntry{
		mockDirEntry{name: "image.png", isDir: false},
	}
	ds.SetDir("/test", entries)

	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	rowCount := ds.ExtTable.GetRowCount()
	ds.ExtTable.Select(rowCount-1, 0)
	res := ds.InputCapture(down)
	assert.Equal(t, down, res)
}

func TestDirSummary_InputCapture_AllBranches(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	groupOne := &viewers.ExtensionsGroup{ExtStats: []*viewers.ExtStat{{ID: ".a"}}}
	groupTwo := &viewers.ExtensionsGroup{ExtStats: []*viewers.ExtStat{{ID: ".a"}, {ID: ".b"}}}

	setRef := func(row int, ref interface{}) {
		cell := tview.NewTableCell("row")
		cell.SetReference(ref)
		ds.ExtTable.SetCell(row, 1, cell)
	}

	keyDown := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	keyUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, "b")
	ds.ExtTable.Select(1, 0)
	res := ds.InputCapture(keyDown)
	assert.Equal(t, keyDown, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupOne)
	setRef(2, "b")
	ds.ExtTable.Select(0, 0)
	res = ds.InputCapture(keyDown)
	assert.Nil(t, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupTwo)
	setRef(2, "b")
	ds.ExtTable.Select(0, 0)
	res = ds.InputCapture(keyDown)
	assert.Equal(t, keyDown, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, "b")
	ds.ExtTable.Select(0, 0)
	res = ds.InputCapture(keyDown)
	assert.Equal(t, keyDown, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	ds.ExtTable.Select(0, 0)
	res = ds.InputCapture(keyUp)
	assert.Equal(t, keyUp, res)

	ds.ExtTable.Clear()
	setRef(0, groupOne)
	setRef(1, "a")
	ds.ExtTable.Select(1, 0)
	res = ds.InputCapture(keyUp)
	assert.Nil(t, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupOne)
	setRef(2, "b")
	ds.ExtTable.Select(2, 0)
	res = ds.InputCapture(keyUp)
	assert.Nil(t, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupTwo)
	setRef(2, "b")
	ds.ExtTable.Select(2, 0)
	res = ds.InputCapture(keyUp)
	assert.Equal(t, keyUp, res)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, "b")
	setRef(2, "c")
	ds.ExtTable.Select(2, 0)
	res = ds.InputCapture(keyUp)
	assert.Equal(t, keyUp, res)
}

func TestNavigator_GetGitStatus_NoRepoRoot(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tempPath := t.TempDir()
	status := nav.getGitStatus(context.Background(), nil, tempPath, true)
	assert.Nil(t, status)
}

func TestFilesPanel_SelectionChangedNavFunc_NoRef(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestDirSummary_InputCapture_Default(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	key := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	res := ds.InputCapture(key)
	assert.Equal(t, key, res)
}

func TestGeneratedNestedDirs_WaitGroup(t *testing.T) {
	store := &mockStoreWithHooks{}
	ctx := context.Background()
	err := GeneratedNestedDirs(ctx, store, "/tmp", "Dir%d", 1, 1)
	assert.NoError(t, err)
}

func TestGeneratedNestedDirs_SubdirError(t *testing.T) {
	store := &failingStore{failOn: "Directory0/Directory0"}
	ctx := context.Background()
	err := GeneratedNestedDirs(ctx, store, "/tmp", "", 2, 1)
	assert.NoError(t, err)
	assert.Greater(t, store.calls, 1)
}

func TestNewPanel_InputCapture_ReturnsEvent(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	panel := NewNewPanel(nav)

	event := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt)
	inputCapture := panel.input.GetInputCapture()
	res := inputCapture(event)
	assert.Equal(t, event, res)
}

func TestNavigator_ShowNewPanel_Focus(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.showNewPanel()
}

func TestTree_SetCurrentDir_Root(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)
	nav.store = mockStore{root: url.URL{Path: "/"}}
	tree.setCurrentDir("/")
}

func TestTree_SetCurrentDir_NonSlashRoot(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)
	nav.store = mockStore{root: url.URL{Path: "/root/"}}
	tree.setCurrentDir("/root/")
}

func TestDirSummary_GetSizes_TypedNilInfo(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)

	var typedNil *nilFileInfo
	entries := []os.DirEntry{
		mockDirEntryInfo{name: "typednil.txt", info: typedNil},
	}
	ds.SetDir("/test", entries)
	err := ds.GetSizes()
	assert.NoError(t, err)
}

func TestFilesPanel_SelectionChangedNavFunc_RefNilAgain(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	cell := tview.NewTableCell("file")
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestNavigator_GetGitStatus_CancelledBeforeStatus(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	oldGetFileStatus := getFileStatus
	defer func() {
		getFileStatus = oldGetFileStatus
	}()

	getFileStatus = func(ctx context.Context, repo *git.Repository, filePath string) *gitutils.RepoStatus {
		_, _, _ = ctx, repo, filePath
		return &gitutils.RepoStatus{Branch: "main"}
	}

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	status := nav.getGitStatus(cancelCtx, repo, filepath.Join(repoDir, "file"), false)
	assert.Nil(t, status)
}

func TestFilesPanel_UpdateGitStatuses_WaitGroup(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.gitStatusCache = make(map[string]*gitutils.RepoStatus)
	fp := newFiles(nav)

	repoDir := t.TempDir()
	repo, initErr := git.PlainInit(repoDir, false)
	assert.NoError(t, initErr)
	assert.NotNil(t, repo)

	filePath := filepath.Join(repoDir, "file.txt")
	writeErr := os.WriteFile(filePath, []byte("content"), 0644)
	assert.NoError(t, writeErr)

	dirContext := &DirContext{Path: repoDir}
	rows := NewFileRows(dirContext)
	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("file.txt", false),
		Dir:      repoDir,
	}
	rows.AllEntries = []files.EntryWithDirPath{entry}
	rows.VisibleEntries = rows.AllEntries
	fp.rows = rows
	fp.table.SetContent(rows)
	nav.store = mockStore{root: url.URL{Scheme: "file", Path: "/"}}

	var mu sync.Mutex
	updated := false
	nav.queueUpdateDraw = func(f func()) {
		f()
		mu.Lock()
		updated = true
		mu.Unlock()
	}

	fp.updateGitStatuses(context.Background(), dirContext)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	assert.True(t, updated)
	mu.Unlock()
}

func TestFilesPanel_SelectionChangedNavFunc_WithRef(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := nav.files

	modTime := files.ModTime(time.Now())
	dirEntry := files.NewDirEntry("file.txt", false, files.Size(1), modTime)
	entry := files.EntryWithDirPath{DirEntry: dirEntry, Dir: "/tmp"}
	cell := tview.NewTableCell("file")
	cell.SetReference(&entry)
	fp.table.SetCell(1, 0, cell)
	fp.selectionChangedNavFunc(1, 0)
}

func TestDirSummary_InputCapture_SkipGroupWithMultipleExt(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newTestDirSummary(nav)
	nav.files = newFiles(nav)
	nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

	entries := []os.DirEntry{
		mockDirEntry{name: "a.go", isDir: false},
		mockDirEntry{name: "b.js", isDir: false},
		mockDirEntry{name: "c.png", isDir: false},
		mockDirEntry{name: "d.jpg", isDir: false},
	}
	ds.SetDir("/test", entries)

	ds.ExtTable.Select(1, 0)
	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	res := ds.InputCapture(down)
	assert.Equal(t, down, res)
}
