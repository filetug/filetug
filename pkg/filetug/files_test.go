package filetug

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func setupNavigatorForFilesTest(app *tview.Application) *Navigator {
	nav := &Navigator{
		app: app,
		setAppFocus: func(p tview.Primitive) {
			app.SetFocus(p)
		},
	}
	nav.right = NewContainer(2, nav)
	nav.previewer = newPreviewerPanel(nav)
	nav.dirsTree = &Tree{tv: tview.NewTreeView()}
	return nav
}

type trackingStore struct {
	root        url.URL
	readDirPath string
	entries     map[string][]os.DirEntry
}

func (t *trackingStore) RootTitle() string { return "Mock" }
func (t *trackingStore) RootURL() url.URL  { return t.root }
func (t *trackingStore) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	_, _ = ctx, name
	t.readDirPath = name
	entries := t.entries[name]
	return entries, nil
}
func (t *trackingStore) CreateDir(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}
func (t *trackingStore) CreateFile(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}
func (t *trackingStore) Delete(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}

func TestNewFiles(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)

	fp := newFiles(nav)
	assert.NotNil(t, fp)
	assert.NotNil(t, fp.table)
	assert.Equal(t, nav, fp.nav)
}

func TestFilesPanel_SetRows(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	dir := &DirContext{Path: "/test"}
	rows := NewFileRows(dir)

	fp.SetRows(rows, true)
	assert.Equal(t, rows, fp.rows)
	// assert.Equal(t, rows, fp.table.GetContent()) // GetContent doesn't exist in tview.Table
	assert.True(t, fp.filter.ShowDirs)
}

func TestFilesPanel_SetFilter(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)
	fp.rows = NewFileRows(&DirContext{})

	filter := ftui.Filter{ShowHidden: true}
	fp.SetFilter(filter)
}

func TestFilesPanel_Selection(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.current.dir = "/test"
	fp := newFiles(nav)

	entries := []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "file1.txt", isDir: false}},
		{DirEntry: mockDirEntry{name: "file2.txt", isDir: false}},
	}
	rows := NewFileRows(&DirContext{Path: "/test"})
	rows.AllEntries = entries
	rows.VisibleEntries = entries
	rows.VisualInfos = make([]os.FileInfo, len(entries))
	fp.rows = rows
	fp.table.SetContent(rows)

	t.Run("SetCurrentFile", func(t *testing.T) {
		fp.SetCurrentFile("file2.txt")
		assert.Equal(t, "file2.txt", fp.currentFileName)
		row, col := fp.table.GetSelection()
		assert.Equal(t, 2, row) // row 0 is .., row 1 is file1, row 2 is file2
		assert.Equal(t, 0, col)
	})

	t.Run("focus_blur", func(t *testing.T) {
		fp.focus()
		assert.Equal(t, 1, nav.activeCol)

		fp.blur()
		// just check it doesn't panic
	})
}

func TestFilesPanel_InputCapture(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	t.Run("Space_Toggle", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)

		cell := tview.NewTableCell(" file1.txt")
		fp.table.SetCell(1, 0, cell)
		fp.table.Select(1, 0)

		res := fp.inputCapture(event)
		assert.Nil(t, res)
		cell = fp.table.GetCell(1, 0)
		assert.True(t, strings.HasPrefix(cell.Text, "✓"), "Expected cell text %q to start with ✓", cell.Text)

		fp.inputCapture(event)
		cell = fp.table.GetCell(1, 0)
		assert.True(t, strings.HasPrefix(cell.Text, " "), "Expected cell text %q to start with space", cell.Text)
	})

	// For other tests that might need rows
	entries := []files.EntryWithDirPath{
		{DirEntry: mockDirEntry{name: "file1.txt", isDir: false}},
		{DirEntry: mockDirEntry{name: "dir1", isDir: true}},
	}
	rows := NewFileRows(&DirContext{Path: "/test"})
	rows.AllEntries = entries
	rows.VisibleEntries = entries
	rows.VisualInfos = make([]os.FileInfo, len(entries))
	fp.rows = rows
	fp.table.SetContent(rows)
	fp.table.Select(1, 0) // Select file1.txt

	t.Run("KeyRight", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Nil(t, res)
	})

	t.Run("KeyUp_TopRow", func(t *testing.T) {
		fp.table.Select(0, 0)
		var moveFocusUpCalled bool
		nav.o.moveFocusUp = func(p tview.Primitive) {
			moveFocusUpCalled = true
		}
		event := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Nil(t, res)
		assert.True(t, moveFocusUpCalled)
	})

	t.Run("KeyUp_TopRow_NoHandler", func(t *testing.T) {
		fp.table.Select(0, 0)
		nav.o.moveFocusUp = nil
		event := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyUp_NotTopRow", func(t *testing.T) {
		fp.table.Select(1, 0)
		event := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyLeft", func(t *testing.T) {
		fp.nav.setAppFocus = func(p tview.Primitive) {
			_, _ = p, p
		}
		event := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Nil(t, res)
	})

	t.Run("KeyDown_Default", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyEnter_Dir", func(t *testing.T) {
		fp.table.Select(2, 0) // Select dir1

		// To avoid panic in Tree.setCurrentDir -> highlightTreeNodes -> node.GetReference()
		// we should ideally mock goDir on Navigator if it was a function,
		// but it's a method. We'll use a real Navigator for this test if possible,
		// or just accept it might be hard to test this branch without a full setup.
		// For now, let's use NewNavigator to get a properly initialized Tree.
		fullNav := NewNavigator(app)
		fp.nav = fullNav

		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		fp.inputCapture(event)
		// res := fp.inputCapture(event)
		// assert.Nil(t, res)
	})

	t.Run("KeyEnter_NoCell", func(t *testing.T) {
		fp.rows = nil
		fp.table.SetContent(nil)
		fp.table.Clear()
		fp.table.Select(0, 0)
		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyEnter_BadRefType", func(t *testing.T) {
		cell := tview.NewTableCell("bad")
		cell.SetReference("not-an-entry")
		fp.table.SetCell(0, 0, cell)
		fp.table.Select(0, 0)
		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyEnter_FileEntry", func(t *testing.T) {
		dir := &DirContext{
			Store: &mockStoreWithHooks{root: url.URL{Scheme: "file", Path: "/"}},
			Path:  "/tmp",
		}
		fp.rows = NewFileRows(dir)
		entry := files.NewEntryWithDirPath(files.NewDirEntry("file.txt", false), "/tmp")
		cell := tview.NewTableCell("file.txt")
		cell.SetReference(entry)
		fp.table.SetCell(0, 0, cell)
		fp.table.Select(0, 0)
		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("KeyEnter_SymlinkDir", func(t *testing.T) {
		tempDir := t.TempDir()
		targetDir := filepath.Join(tempDir, "target")
		err := os.Mkdir(targetDir, 0o755)
		if !assert.NoError(t, err) {
			return
		}
		linkPath := filepath.Join(tempDir, "link")
		err = os.Symlink(targetDir, linkPath)
		if !assert.NoError(t, err) {
			return
		}
		entries, err := os.ReadDir(tempDir)
		if !assert.NoError(t, err) {
			return
		}
		var linkEntry os.DirEntry
		for _, entry := range entries {
			if entry.Name() == "link" {
				linkEntry = entry
				break
			}
		}
		if !assert.NotNil(t, linkEntry) {
			return
		}
		fullNav := NewNavigator(app)
		fp.nav = fullNav
		fp.rows = NewFileRows(&DirContext{
			Store: &mockStoreWithHooks{root: url.URL{Scheme: "file", Path: "/"}},
			Path:  tempDir,
		})
		entry := files.EntryWithDirPath{DirEntry: linkEntry, Dir: tempDir}
		cell := tview.NewTableCell("link")
		cell.SetReference(&entry)
		fp.table.SetCell(0, 0, cell)
		fp.table.Select(0, 0)
		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := fp.inputCapture(event)
		assert.Nil(t, res)
	})
}

func TestFilesPanel_SelectionChanged(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.current.dir = "/test"
	nav.dirSummary = newTestDirSummary(nav)

	fp := newFiles(nav)
	nav.files = fp

	store := &trackingStore{
		root: url.URL{Scheme: "file", Path: "/"},
		entries: map[string][]os.DirEntry{
			"/":           {mockDirEntry{name: "test", isDir: true}},
			"/test/child": {mockDirEntry{name: "file.txt", isDir: false}},
		},
	}
	nav.store = store

	entries := []files.EntryWithDirPath{
		{DirEntry: files.NewDirEntry("child", true), Dir: "/test"},
	}
	rows := NewFileRows(&DirContext{Store: store, Path: "/test"})
	rows.AllEntries = entries
	rows.VisibleEntries = entries
	rows.VisualInfos = make([]os.FileInfo, len(entries))
	fp.rows = rows
	fp.table.SetContent(rows)

	// Test row 0 (parent dir)
	fp.selectionChanged(0, 0)
	assert.Equal(t, nav.dirSummary, nav.right.content)
	assert.Equal(t, "/", store.readDirPath)

	// Test dir row
	fp.selectionChanged(1, 0)
	assert.Equal(t, nav.dirSummary, nav.right.content)
	assert.Equal(t, "/test/child", store.readDirPath)
}

func TestFilesPanel_OnStoreChange(t *testing.T) {
	app := tview.NewApplication()
	nav := &Navigator{
		app: app,
		setAppFocus: func(p tview.Primitive) {
			app.SetFocus(p)
		},
	}
	fp := newFiles(nav)

	fp.onStoreChange()
	assert.Equal(t, 0, fp.loadingProgress)
	assert.Equal(t, "Loading...", fp.table.GetCell(0, 0).Text)
}

func TestFilesPanel_selectionChangedNavFunc(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	fp.table.SetCell(1, 0, tview.NewTableCell(" file1.txt"))
	fp.selectionChangedNavFunc(1, 0)
}

type errorReadDirStore struct {
	root        url.URL
	readDirPath string
	readErr     error
}

func (e *errorReadDirStore) RootTitle() string { return "Mock" }
func (e *errorReadDirStore) RootURL() url.URL  { return e.root }
func (e *errorReadDirStore) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	_, _ = ctx, name
	e.readDirPath = name
	return nil, e.readErr
}
func (e *errorReadDirStore) CreateDir(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}
func (e *errorReadDirStore) CreateFile(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}
func (e *errorReadDirStore) Delete(ctx context.Context, p string) error {
	_, _ = ctx, p
	return nil
}

func TestFilesPanel_entryFromRow_MissingData(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	fp := newFiles(nav)

	fp.table = nil
	entry := fp.entryFromRow(0)
	assert.Nil(t, entry)

	fp.table = tview.NewTable()
	noRefCell := tview.NewTableCell("no ref")
	fp.table.SetCell(0, 0, noRefCell)
	entry = fp.entryFromRow(0)
	assert.Nil(t, entry)

	badRefCell := tview.NewTableCell("bad ref")
	badRefCell.SetReference("not-an-entry")
	fp.table.SetCell(0, 0, badRefCell)
	entry = fp.entryFromRow(0)
	assert.Nil(t, entry)

	nilRefCell := tview.NewTableCell("nil ref")
	var nilEntry files.EntryWithDirPath
	nilRefCell.SetReference(nilEntry)
	fp.table.SetCell(0, 0, nilRefCell)
	entry = fp.entryFromRow(0)
	assert.Nil(t, entry)
}

func TestFilesPanel_updatePreviewForEntry_FileNoPreviewer(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.previewer = nil
	nav.right = NewContainer(2, nav)
	fp := newFiles(nav)

	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("file.txt", false),
		Dir:      "/tmp",
	}
	fp.updatePreviewForEntry(entry)
	assert.Equal(t, "file.txt", fp.currentFileName)
	assert.Nil(t, nav.right.content)
}

func TestFilesPanel_updatePreviewForEntry_FileWithPreviewer(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.right = NewContainer(2, nav)
	nav.previewer = newPreviewerPanel(nav)
	fp := newFiles(nav)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	err := os.WriteFile(filePath, []byte("content"), 0o644)
	if !assert.NoError(t, err) {
		return
	}
	entries, err := os.ReadDir(tempDir)
	if !assert.NoError(t, err) {
		return
	}
	var fileEntry os.DirEntry
	for _, entry := range entries {
		if entry.Name() == "file.txt" {
			fileEntry = entry
			break
		}
	}
	if !assert.NotNil(t, fileEntry) {
		return
	}

	entry := files.EntryWithDirPath{
		DirEntry: fileEntry,
		Dir:      tempDir,
	}
	fp.updatePreviewForEntry(entry)
	assert.Equal(t, nav.previewer, nav.right.content)
}

func TestFilesPanel_updatePreviewForEntry_Dir(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.right = NewContainer(2, nav)
	nav.dirSummary = newTestDirSummary(nav)
	fp := newFiles(nav)

	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("dir", true),
		Dir:      "/tmp",
	}
	fp.updatePreviewForEntry(entry)
	assert.Equal(t, nav.dirSummary, nav.right.content)
}

func TestFilesPanel_updatePreviewForEntry_NoNav(t *testing.T) {
	fp := &filesPanel{}
	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("file.txt", false),
		Dir:      "/tmp",
	}
	fp.updatePreviewForEntry(entry)
}

func TestFilesPanel_showDirSummary_StoreNil(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.right = NewContainer(2, nav)
	nav.dirSummary = newTestDirSummary(nav)
	fp := newFiles(nav)

	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("dir", true),
		Dir:      "/tmp",
	}
	fp.showDirSummary(entry)
	assert.Equal(t, nav.dirSummary, nav.right.content)
	assert.Len(t, nav.dirSummary.ExtStats, 0)
}

func TestFilesPanel_showDirSummary_ReadDirError(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.right = NewContainer(2, nav)
	nav.dirSummary = newTestDirSummary(nav)
	store := &errorReadDirStore{
		root:    url.URL{Scheme: "file", Path: "/"},
		readErr: assert.AnError,
	}
	nav.store = store
	fp := newFiles(nav)

	entry := files.EntryWithDirPath{
		DirEntry: files.NewDirEntry("dir", true),
		Dir:      "/tmp",
	}
	fp.showDirSummary(entry)
	assert.Equal(t, nav.dirSummary, nav.right.content)
	assert.Len(t, nav.dirSummary.ExtStats, 0)
	assert.Equal(t, "/tmp/dir", store.readDirPath)
}

func TestFilesPanel_showDirSummary_Symlink(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.right = NewContainer(2, nav)
	nav.dirSummary = newTestDirSummary(nav)
	fp := newFiles(nav)

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "target")
	err := os.Mkdir(targetDir, 0o755)
	if !assert.NoError(t, err) {
		return
	}
	linkPath := filepath.Join(tempDir, "link")
	err = os.Symlink(targetDir, linkPath)
	if !assert.NoError(t, err) {
		return
	}
	entries, err := os.ReadDir(tempDir)
	if !assert.NoError(t, err) {
		return
	}
	var linkEntry os.DirEntry
	for _, entry := range entries {
		if entry.Name() == "link" {
			linkEntry = entry
			break
		}
	}
	if !assert.NotNil(t, linkEntry) {
		return
	}

	store := &trackingStore{
		root: url.URL{Scheme: "file", Path: "/"},
		entries: map[string][]os.DirEntry{
			linkPath: {},
		},
	}
	nav.store = store
	fp.rows = NewFileRows(&DirContext{Store: store, Path: tempDir})

	entry := files.EntryWithDirPath{
		DirEntry: linkEntry,
		Dir:      tempDir,
	}
	fp.showDirSummary(entry)
	assert.Equal(t, linkPath, store.readDirPath)
}
