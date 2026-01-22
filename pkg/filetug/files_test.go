package filetug

import (
	"os"
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
	nav.right = newContainer(2, nav)
	nav.previewer = &previewer{textView: tview.NewTextView()}
	nav.dirsTree = &Tree{TreeView: tview.NewTreeView()}
	return nav
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

	entries := []os.DirEntry{
		mockDirEntry{name: "file1.txt", isDir: false},
		mockDirEntry{name: "file2.txt", isDir: false},
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
	entries := []os.DirEntry{
		mockDirEntry{name: "file1.txt", isDir: false},
		mockDirEntry{name: "dir1", isDir: true},
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

	t.Run("KeyUp_NotTopRow", func(t *testing.T) {
		fp.table.Select(1, 0)
		event := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
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
		res := fp.inputCapture(event)
		assert.Nil(t, res)
	})
}

func TestFilesPanel_SelectionChanged(t *testing.T) {
	app := tview.NewApplication()
	nav := setupNavigatorForFilesTest(app)
	nav.current.dir = "/test"
	fp := newFiles(nav)

	entries := []os.DirEntry{
		files.NewDirEntry("file1.txt", false),
	}
	rows := NewFileRows(&DirContext{Path: "/test"})
	rows.VisibleEntries = entries
	rows.VisualInfos = []os.FileInfo{
		files.NewFileInfo(entries[0].(files.DirEntry)),
	}
	fp.rows = rows
	fp.table.SetContent(rows)

	// Test row 0 (parent dir)
	fp.selectionChanged(0, 0)
	assert.Contains(t, nav.previewer.textView.GetText(true), "Selected dir: /test")

	// Test file row
	fp.selectionChanged(1, 0)
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
