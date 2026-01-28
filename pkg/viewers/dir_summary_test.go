package viewers

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type mockFileInfo struct {
	os.FileInfo
	size  int64
	isDir bool
}

func (m mockFileInfo) Size() int64 { return m.size }
func (m mockFileInfo) IsDir() bool { return m.isDir }

type mockDirEntryWithInfo struct {
	mockDirEntry
	info os.FileInfo
	err  error
}

func (m mockDirEntryWithInfo) Info() (os.FileInfo, error) { return m.info, m.err }

func newDirContext(path string, entries []os.DirEntry) *files.DirContext {
	return files.NewDirContext(nil, path, entries)
}

func tabsActiveIndex(t *testing.T, tabs *sneatv.Tabs) int {
	t.Helper()
	value := reflect.ValueOf(tabs).Elem().FieldByName("active")
	return int(value.Int())
}

func TestNewDirSummary(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	assert.NotNil(t, ds)
	assert.NotNil(t, ds.ExtTable)
}

func TestDirSummary_SetDir(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	entries := []os.DirEntry{
		mockDirEntry{name: "image1.png", isDir: false},
		mockDirEntry{name: "image2.png", isDir: false},
		mockDirEntry{name: "script.go", isDir: false},
		mockDirEntry{name: "unknown.foo", isDir: false},
		mockDirEntry{name: "subdir", isDir: true},
		mockDirEntry{name: ".noext", isDir: false},
		mockDirEntry{name: "noext", isDir: false},
		mockDirEntry{name: "data.json", isDir: false},
	}

	dirContext := newDirContext("/test", entries)
	ds.SetDirEntries(dirContext)

	var imageGroup *ExtensionsGroup
	for _, g := range ds.ExtGroups {
		if g.ID == "Image" {
			imageGroup = g
			break
		}
	}
	if imageGroup == nil {
		t.Fatal("expected imageGroup to be not nil")
	}
	assert.Equal(t, "Images", imageGroup.Title)
	assert.Len(t, imageGroup.ExtStats, 1)
}

func TestGetSizeCell(t *testing.T) {
	testCases := []struct {
		size int64
	}{
		{1024 * 1024 * 1024 * 1024 * 2},
		{1024 * 1024 * 1024 * 2},
		{1024 * 1024 * 2},
		{1024 * 2},
		{1024},
		{512},
		{1},
		{0},
		{-1},
	}

	for _, tc := range testCases {
		cell := GetSizeCell(tc.size, tcell.ColorWhite)
		assert.NotEmpty(t, cell.Text)
	}

	s1 := GetSizeCell(1024*1024*1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s1)
	s2 := GetSizeCell(1024*1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s2)
	s3 := GetSizeCell(1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s3)
	s4 := GetSizeCell(1024, tcell.ColorWhite)
	assert.NotNil(t, s4)
	s5 := GetSizeCell(100, tcell.ColorWhite)
	assert.NotNil(t, s5)
	s6 := GetSizeCell(0, tcell.ColorWhite)
	assert.NotNil(t, s6)

	GetSizeCell(1024*1024*1024*1024+1, tcell.ColorWhite)
	GetSizeCell(1024*1024*1024+1, tcell.ColorWhite)
	GetSizeCell(1024*1024+1, tcell.ColorWhite)
	GetSizeCell(1024+1, tcell.ColorWhite)
}

func TestDirSummary_Extra(t *testing.T) {
	app := tview.NewApplication()
	var lastFilter ftui.Filter
	filterSetter := WithDirSummaryFilterSetter(func(filter ftui.Filter) {
		lastFilter = filter
	})
	focusCalled := false
	focusLeft := WithDirSummaryFocusLeft(func() {
		focusCalled = true
	})
	ds := NewDirSummary(app, filterSetter, focusLeft)

	t.Run("Focus", func(t *testing.T) {
		ds.Focus(func(p tview.Primitive) {
			app.SetFocus(p)
		})
	})

	t.Run("selectionChanged", func(t *testing.T) {
		entries := []os.DirEntry{
			mockDirEntry{name: "image1.png", isDir: false},
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		dirContext := newDirContext("/test/selection", entries)
		ds.SetDirEntries(dirContext)

		ds.selectionChanged(1, 0)
		assert.Len(t, lastFilter.Extensions, 1)

		ds.selectionChanged(0, 0)
		ds.selectionChanged(-1, 0)
	})

	t.Run("inputCapture", func(t *testing.T) {
		entriesSkip := []os.DirEntry{
			mockDirEntry{name: "image1.png", isDir: false},
			mockDirEntry{name: "image2.jpg", isDir: false},
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		dirContext := newDirContext("/test/input", entriesSkip)
		ds.SetDirEntries(dirContext)

		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		assert.Nil(t, ds.inputCapture(eventLeft))
		assert.True(t, focusCalled)

		eventDown := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		ds.ExtTable.Select(2, 0)
		res := ds.inputCapture(eventDown)
		assert.Nil(t, res)
		row, _ := ds.ExtTable.GetSelection()
		assert.Equal(t, 4, row)

		ds.ExtTable.Select(4, 0)
		assert.NotNil(t, ds.inputCapture(eventDown))

		eventUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		ds.ExtTable.Select(4, 0)
		res = ds.inputCapture(eventUp)
		assert.Nil(t, res)
		row, _ = ds.ExtTable.GetSelection()
		assert.Equal(t, 2, row)

		ds.ExtTable.Select(0, 0)
		assert.NotNil(t, ds.inputCapture(eventUp))

		entriesSingle := []os.DirEntry{
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		dirContext = newDirContext("/test/input/single", entriesSingle)
		ds.SetDirEntries(dirContext)
		ds.ExtTable.Select(1, 0)
		assert.Nil(t, ds.inputCapture(eventUp))

		eventOther := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
		assert.Equal(t, eventOther, ds.inputCapture(eventOther))
	})

	t.Run("GetSizes", func(t *testing.T) {
		entries := []os.DirEntry{
			mockDirEntryWithInfo{
				mockDirEntry: mockDirEntry{name: "image1.png", isDir: false},
				info:         mockFileInfo{size: 100},
			},
			mockDirEntryWithInfo{
				mockDirEntry: mockDirEntry{name: "error.png", isDir: false},
				err:          assert.AnError,
			},
			mockDirEntryWithInfo{
				mockDirEntry: mockDirEntry{name: "nil.png", isDir: false},
				info:         nil,
			},
		}
		dirContext := newDirContext("/test/sizes", entries)
		ds.SetDirEntries(dirContext)
		err := ds.GetSizes()
		assert.Error(t, err)
	})
}

func TestDirSummary_PreviewAndOptions(t *testing.T) {
	app := tview.NewApplication()
	queueCalled := false
	queueUpdate := WithDirSummaryQueueUpdateDraw(func(f func()) {
		queueCalled = true
		f()
	})
	colorByExt := WithDirSummaryColorByExt(func(_ string) tcell.Color {
		return tcell.ColorBlue
	})
	ds := NewDirSummary(app, queueUpdate, colorByExt)

	assert.NotNil(t, ds.queueUpdateDraw)
	assert.NotNil(t, ds.colorByExt)

	tempDir := t.TempDir()
	filePath := tempDir + "/a.txt"
	writeErr := os.WriteFile(filePath, []byte("hello"), 0644)
	assert.NoError(t, writeErr)

	entry := files.NewEntryWithDirPath(mockDirEntry{name: "temp", isDir: true}, tempDir)
	ds.Preview(entry, nil, nil, ds.queueUpdateDraw)

	ds.queueUpdate(func() {})
	assert.True(t, queueCalled)
	assert.Equal(t, ds, ds.Main())
	assert.Nil(t, ds.Meta())
}

func TestDirSummary_InputCapture_LeftWithoutFocus(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	entries := []os.DirEntry{
		mockDirEntry{name: "image.png", isDir: false},
	}
	dirContext := newDirContext("/test", entries)
	ds.SetDirEntries(dirContext)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res := ds.InputCapture(left)
	assert.Equal(t, left, res)
}

func TestDirSummary_SetDir_WithRepo(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}

	tempDir := t.TempDir()
	gitDir := tempDir + "/.git"
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}
	dirContext := newDirContext(tempDir, entries)
	ds.SetDirEntries(dirContext)
	assert.NotNil(t, ds.tabs)
}

func TestDirSummary_SetDir_ActivateGitTabWhenDirty(t *testing.T) {
	app := tview.NewApplication()
	updates := make(chan struct{}, 5)
	ds := NewDirSummary(app, WithDirSummaryQueueUpdateDraw(func(f func()) {
		f()
		updates <- struct{}{}
	}))
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{
			repoRoot: "/repo",
			entries: []gitDirStatusEntry{
				{displayName: "a.txt"},
			},
		}, nil
	}

	tempDir := t.TempDir()
	gitDir := tempDir + "/.git"
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}
	dirContext := newDirContext(tempDir, entries)
	ds.SetDirEntries(dirContext)

	deadline := time.After(time.Second)
	for {
		if tabsActiveIndex(t, ds.tabs) == 1 {
			return
		}
		select {
		case <-updates:
		case <-deadline:
			t.Fatal("expected git tab to be active")
		}
	}
}

func TestDirSummary_SetDir_DirtyGitTabNotActivated(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app, WithDirSummaryQueueUpdateDraw(func(f func()) {
		f()
	}))
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}

	tempDir := t.TempDir()
	gitDir := tempDir + "/.git"
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}
	dirContext := newDirContext(tempDir, entries)
	ds.SetDirEntries(dirContext)

	activeIndex := tabsActiveIndex(t, ds.tabs)
	assert.Equal(t, 0, activeIndex)
}

func TestDirSummary_SetDir_StatusLoaderNil(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.GitPreviewer.statusLoader = nil

	tempDir := t.TempDir()
	gitDir := tempDir + "/.git"
	mkdirErr := os.Mkdir(gitDir, 0755)
	assert.NoError(t, mkdirErr)

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}
	dirContext := newDirContext(tempDir, entries)
	ds.SetDirEntries(dirContext)
}

func TestDirSummary_ActivateGitTabIfDirty_GitPreviewerNil(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.GitPreviewer = nil

	ds.activateGitTabIfDirty("/tmp")
}

func TestDirSummary_ActivateGitTabIfDirty_StatusLoaderError(t *testing.T) {
	app := tview.NewApplication()
	loaderCalled := make(chan struct{}, 1)
	ds := NewDirSummary(app)
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		loaderCalled <- struct{}{}
		return gitDirStatusResult{}, assert.AnError
	}

	ds.activateGitTabIfDirty("/tmp")

	select {
	case <-loaderCalled:
	case <-time.After(time.Second):
		t.Fatal("expected status loader to be called")
	}
}

func TestDirSummary_ActivateGitTabIfDirty_NoRepoRoot(t *testing.T) {
	app := tview.NewApplication()
	loaderCalled := make(chan struct{}, 1)
	ds := NewDirSummary(app)
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		loaderCalled <- struct{}{}
		return gitDirStatusResult{}, nil
	}

	ds.activateGitTabIfDirty("/tmp")

	select {
	case <-loaderCalled:
	case <-time.After(time.Second):
		t.Fatal("expected status loader to be called")
	}
}

func TestDirSummary_ActivateGitTabIfDirty_EmptyEntries(t *testing.T) {
	app := tview.NewApplication()
	loaderCalled := make(chan struct{}, 1)
	ds := NewDirSummary(app)
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		loaderCalled <- struct{}{}
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}

	ds.activateGitTabIfDirty("/tmp")

	select {
	case <-loaderCalled:
	case <-time.After(time.Second):
		t.Fatal("expected status loader to be called")
	}
}

func TestDirSummary_ActivateGitTabIfDirty_TabsNil(t *testing.T) {
	app := tview.NewApplication()
	updateCalled := make(chan struct{}, 1)
	ds := NewDirSummary(app, WithDirSummaryQueueUpdateDraw(func(f func()) {
		f()
		updateCalled <- struct{}{}
	}))
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{
			repoRoot: "/repo",
			entries: []gitDirStatusEntry{
				{displayName: "a.txt"},
			},
		}, nil
	}
	ds.tabs = nil
	ds.dirPath = "/tmp"

	ds.activateGitTabIfDirty("/tmp")

	select {
	case <-updateCalled:
	case <-time.After(time.Second):
		t.Fatal("expected queue update to be called")
	}
}

func TestDirSummary_ActivateGitTabIfDirty_DirPathChanged(t *testing.T) {
	app := tview.NewApplication()
	updateCalled := make(chan struct{}, 1)
	releaseLoader := make(chan struct{})
	ds := NewDirSummary(app, WithDirSummaryQueueUpdateDraw(func(f func()) {
		f()
		updateCalled <- struct{}{}
	}))
	ds.GitPreviewer.statusLoader = func(_ string) (gitDirStatusResult, error) {
		<-releaseLoader
		return gitDirStatusResult{
			repoRoot: "/repo",
			entries: []gitDirStatusEntry{
				{displayName: "a.txt"},
			},
		}, nil
	}
	ds.dirPath = "/tmp"

	ds.activateGitTabIfDirty("/tmp")
	ds.dirPath = "/other"
	close(releaseLoader)

	select {
	case <-updateCalled:
	case <-time.After(time.Second):
		t.Fatal("expected queue update to be called")
	}
}

func TestDirSummary_UpdateTable_NoQueue(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.queueUpdateDraw = nil
	ds.ExtGroups = []*ExtensionsGroup{
		{
			ID:    "Text",
			Title: "Texts",
			GroupStats: &GroupStats{
				Count:     1,
				TotalSize: 1,
			},
			ExtStats: []*ExtStat{
				{
					ID: ".txt",
					GroupStats: GroupStats{
						Count:     1,
						TotalSize: 1,
					},
				},
			},
		},
	}
	ds.UpdateTable()
	cell := ds.ExtTable.GetCell(1, 1)
	assert.Contains(t, cell.Text, ".txt")
}

func TestDirSummary_Preview_FileEntryAndError(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	tempDir := t.TempDir()
	filePath := tempDir + "/b.log"
	writeErr := os.WriteFile(filePath, []byte("log"), 0644)
	assert.NoError(t, writeErr)

	entries, err := os.ReadDir(tempDir)
	assert.NoError(t, err)
	dirContext := newDirContext(tempDir, entries)
	ds.SetDirEntries(dirContext)
	assert.NotEmpty(t, ds.ExtGroups)

	fileEntry := files.NewEntryWithDirPath(mockDirEntry{name: "b.log", isDir: false}, tempDir)
	ds.Preview(fileEntry, nil, nil, nil)

	badEntry := files.NewEntryWithDirPath(mockDirEntry{name: "missing", isDir: true}, tempDir+"/nope")
	ds.Preview(badEntry, nil, nil, nil)
}

func TestDirSummary_Preview_DirContext(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	dirContext := files.NewDirContext(nil, "/test", []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	})

	ds.Preview(dirContext, nil, nil, nil)
	assert.NotEmpty(t, ds.ExtGroups)
}

func TestDirSummary_QueueUpdate_NoQueue(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.queueUpdateDraw = nil
	called := false
	ds.queueUpdate(func() {
		called = true
	})
	assert.True(t, called)
}

func TestDirSummary_InputCapture_Edges(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}
	dirContext := newDirContext("/test", entries)
	ds.SetDirEntries(dirContext)
	ds.ExtTable.Select(0, 0)

	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	assert.Equal(t, up, ds.InputCapture(up))

	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	rowCount := ds.ExtTable.GetRowCount()
	ds.ExtTable.Select(rowCount-1, 0)
	assert.Equal(t, down, ds.InputCapture(down))
}

func TestDirSummary_SelectionChanged_NoFilterSetter(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)
	ds.ExtTable.SetCell(0, 1, tview.NewTableCell("cell"))
	ds.selectionChanged(0, 0)
}

func TestDirSummary_UpdateTable_MixedGroups(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	ds.ExtGroups = []*ExtensionsGroup{
		{
			ID:    "Single",
			Title: "Singles",
			GroupStats: &GroupStats{
				Count:     1,
				TotalSize: 10,
			},
			ExtStats: []*ExtStat{
				{
					ID: "",
					GroupStats: GroupStats{
						Count:     1,
						TotalSize: 10,
					},
				},
			},
		},
		{
			ID:    "Multi",
			Title: "Multis",
			GroupStats: &GroupStats{
				Count:     3,
				TotalSize: 20,
			},
			ExtStats: []*ExtStat{
				{
					ID: ".a",
					GroupStats: GroupStats{
						Count:     1,
						TotalSize: 5,
					},
				},
				{
					ID: ".b",
					GroupStats: GroupStats{
						Count:     2,
						TotalSize: 15,
					},
				},
			},
		},
	}
	ds.UpdateTable()
	cell := ds.ExtTable.GetCell(1, 1)
	assert.Contains(t, cell.Text, "<no extension>")
}

func TestDirSummary_GetSizes_NilAndTypedNil(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	var typedNil *mockFileInfo
	entries := []os.DirEntry{
		mockDirEntryWithInfo{
			mockDirEntry: mockDirEntry{name: "nil.txt", isDir: false},
			info:         nil,
		},
		mockDirEntryWithInfo{
			mockDirEntry: mockDirEntry{name: "typednil.txt", isDir: false},
			info:         typedNil,
		},
	}
	dirContext := newDirContext("/test", entries)
	ds.SetDirEntries(dirContext)
	err := ds.GetSizes()
	assert.NoError(t, err)
}

func TestDirSummary_InputCapture_Branches(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	groupSingle := &ExtensionsGroup{ExtStats: []*ExtStat{{ID: ".a"}}}
	groupMulti := &ExtensionsGroup{ExtStats: []*ExtStat{{ID: ".a"}, {ID: ".b"}}}

	setRef := func(row int, ref interface{}) {
		cell := tview.NewTableCell("row")
		cell.SetReference(ref)
		ds.ExtTable.SetCell(row, 1, cell)
	}

	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupSingle)
	setRef(2, "b")
	ds.ExtTable.Select(0, 0)
	assert.Nil(t, ds.InputCapture(down))
	row, _ := ds.ExtTable.GetSelection()
	assert.Equal(t, 2, row)

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupMulti)
	setRef(2, "b")
	ds.ExtTable.Select(0, 0)
	assert.Equal(t, down, ds.InputCapture(down))

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, "b")
	ds.ExtTable.Select(0, 0)
	assert.Equal(t, down, ds.InputCapture(down))

	ds.ExtTable.Clear()
	setRef(0, groupSingle)
	setRef(1, "a")
	ds.ExtTable.Select(1, 0)
	assert.Nil(t, ds.InputCapture(up))

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupSingle)
	setRef(2, "b")
	ds.ExtTable.Select(2, 0)
	assert.Nil(t, ds.InputCapture(up))

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, groupMulti)
	setRef(2, "b")
	ds.ExtTable.Select(2, 0)
	assert.Equal(t, up, ds.InputCapture(up))

	ds.ExtTable.Clear()
	setRef(0, "a")
	setRef(1, "b")
	ds.ExtTable.Select(1, 0)
	assert.Equal(t, up, ds.InputCapture(up))
}

func TestDirSummary_SetDir_GetSizesErrorInQueue(t *testing.T) {
	app := tview.NewApplication()
	queueUpdate := WithDirSummaryQueueUpdateDraw(func(f func()) { f() })
	ds := NewDirSummary(app, queueUpdate)

	entries := []os.DirEntry{
		mockDirEntryWithInfo{
			mockDirEntry: mockDirEntry{name: "bad.txt", isDir: false},
			err:          assert.AnError,
		},
	}
	dirContext := newDirContext("/test", entries)
	ds.SetDirEntries(dirContext)
}

func TestDirSummary_UpdateTable_SingleCountForGroup(t *testing.T) {
	app := tview.NewApplication()
	ds := NewDirSummary(app)

	ds.ExtGroups = []*ExtensionsGroup{
		{
			ID:    "Multi",
			Title: "Multis",
			GroupStats: &GroupStats{
				Count:     1,
				TotalSize: 3,
			},
			ExtStats: []*ExtStat{
				{
					ID: ".a",
					GroupStats: GroupStats{
						Count:     1,
						TotalSize: 1,
					},
				},
				{
					ID: ".b",
					GroupStats: GroupStats{
						Count:     2,
						TotalSize: 2,
					},
				},
			},
		},
	}
	ds.UpdateTable()
	cell := ds.ExtTable.GetCell(0, 2)
	assert.Contains(t, cell.Text, "1")
}

func TestDirSummary_SetDirEntries_StaleQueueUpdate(t *testing.T) {
	app := tview.NewApplication()
	queueUpdates := make(chan func(), 2)
	ds := NewDirSummary(app, WithDirSummaryQueueUpdateDraw(func(f func()) {
		queueUpdates <- f
	}))

	entries := []os.DirEntry{
		mockDirEntry{name: "a.txt", isDir: false},
	}

	dirContext := newDirContext("/first", entries)
	ds.SetDirEntries(dirContext)
	var firstUpdate func()
	select {
	case firstUpdate = <-queueUpdates:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first queue update")
	}

	dirContext = newDirContext("/second", entries)
	ds.SetDirEntries(dirContext)
	firstUpdate()

	select {
	case nextUpdate := <-queueUpdates:
		nextUpdate()
	default:
	}
}
