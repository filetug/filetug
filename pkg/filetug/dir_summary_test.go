package filetug

import (
	"os"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewDirSummary(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newDirSummary(nav)
	assert.NotNil(t, ds)
	assert.NotNil(t, ds.extTable)
}

type mockFileInfo struct {
	os.FileInfo
	size int64
}

func (m mockFileInfo) Size() int64 { return m.size }

type mockDirEntryWithInfo struct {
	mockDirEntry
	info os.FileInfo
	err  error
}

func (m mockDirEntryWithInfo) Info() (os.FileInfo, error) { return m.info, m.err }

func TestDirSummary_SetDir(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newDirSummary(nav)

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

	dir := &DirContext{
		Path:     "/test",
		children: entries,
	}

	ds.SetDir(dir)

	// .png -> Image, .go -> Code, .foo -> Other, .noext -> Other, noext -> (skipped if extID == name)
	// Wait, "noext" will have extID = "" which is != "noext", so it's NOT skipped.
	// path.Ext("noext") is ""
	// path.Ext(".noext") is ".noext" -> this matches extID == name and IS skipped.

	var imageGroup *extensionsGroup
	for _, g := range ds.extGroups {
		if g.id == "Image" {
			imageGroup = g
			break
		}
	}
	if imageGroup == nil {
		t.Fatal("expected imageGroup to be not nil")
	}
	assert.Equal(t, "Images", imageGroup.title)
	assert.Len(t, imageGroup.extStats, 1) // .png
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
		cell := getSizeCell(tc.size, tcell.ColorWhite)
		assert.NotEmpty(t, cell.Text)
	}

	// Specifically test the thresholds to ensure coverage
	s1 := getSizeCell(1024*1024*1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s1)
	s2 := getSizeCell(1024*1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s2)
	s3 := getSizeCell(1024*1024, tcell.ColorWhite)
	assert.NotNil(t, s3)
	s4 := getSizeCell(1024, tcell.ColorWhite)
	assert.NotNil(t, s4)
	s5 := getSizeCell(100, tcell.ColorWhite)
	assert.NotNil(t, s5)
	s6 := getSizeCell(0, tcell.ColorWhite)
	assert.NotNil(t, s6)

	// Extra thresholds for coverage
	getSizeCell(1024*1024*1024*1024+1, tcell.ColorWhite)
	getSizeCell(1024*1024*1024+1, tcell.ColorWhite)
	getSizeCell(1024*1024+1, tcell.ColorWhite)
	getSizeCell(1024+1, tcell.ColorWhite)
}

func TestDirSummary_Extra(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	if nav == nil {
		t.Fatal("expected navigator to be not nil")
	}
	nav.files = newFiles(nav) // Ensure nav.files is initialized to avoid panic
	ds := newDirSummary(nav)

	t.Run("Focus", func(t *testing.T) {
		ds.Focus(func(p tview.Primitive) {
			app.SetFocus(p)
		})
	})

	t.Run("selectionChanged", func(t *testing.T) {
		// Mock data to ensure we have rows
		entries := []os.DirEntry{
			mockDirEntry{name: "image1.png", isDir: false},
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		ds.SetDir(&DirContext{Path: "/test", children: entries})

		// Properly initialize nav.files and its rows to avoid panic in SetFilter
		nav.files.rows = NewFileRows(&DirContext{Path: "/test"})

		// Test extension selection
		ds.selectionChanged(1, 0)

		// Test group selection
		ds.selectionChanged(0, 0)

		// Test negative row
		ds.selectionChanged(-1, 0)
	})

	t.Run("inputCapture", func(t *testing.T) {
		// Mock data with a group that has multiple extensions and a group that has one extension
		entries := []os.DirEntry{
			mockDirEntry{name: "image1.png", isDir: false},
			mockDirEntry{name: "image2.jpg", isDir: false},
			mockDirEntry{name: "script.go", isDir: false},
		}
		ds.SetDir(&DirContext{Path: "/test", children: entries})

		// Rows should be:
		// Row 0: Code (group, 1 ext)
		// Row 1: .go
		// Row 2: Images (group, 2 exts)
		// Row 3: .jpg
		// Row 4: .png

		// Test Left
		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		assert.Nil(t, ds.inputCapture(eventLeft))

		// Test Down skipping
		eventDown := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		// Select row 0 (Code), it has 1 ext, so KeyDown on it should return event (tview will handle selection)
		// Wait, the logic is in inputCapture: if next row is a group with 1 ext, it skips.
		// If we are at row 0, it's NOT next row.
		// If we are at row -1 (nothing selected), and press down, it selects 0.
		// Let's test skipping from row 1 to row 3 because row 2 is a single-ext group? No, Images has 2 exts.
		// Let's re-mock to have:
		// Images (2 exts)
		// .jpg
		// .png
		// Video (1 ext) -> .mp4
		entriesSkip := []os.DirEntry{
			mockDirEntry{name: "image1.png", isDir: false},
			mockDirEntry{name: "image2.jpg", isDir: false},
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		ds.SetDir(&DirContext{Path: "/test", children: entriesSkip})
		// Rows:
		// 0: Images (2 exts)
		// 1: .jpg
		// 2: .png
		// 3: Videos (1 ext)
		// 4: .mp4

		ds.extTable.Select(2, 0) // Select .png
		// Next row (3) is "Videos" group which has 1 ext (.mp4).
		// inputCapture should skip row 3 and select row 4.
		res := ds.inputCapture(eventDown)
		assert.Nil(t, res)
		row, _ := ds.extTable.GetSelection()
		assert.Equal(t, 4, row)

		// Test Down at bottom
		ds.extTable.Select(4, 0)
		assert.NotNil(t, ds.inputCapture(eventDown))

		// Test Up skipping
		eventUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		ds.extTable.Select(4, 0) // Select .mp4
		// Prev row (3) is "Videos" group with 1 ext.
		// inputCapture should skip row 3 and select row 2.
		res = ds.inputCapture(eventUp)
		assert.Nil(t, res)
		row, _ = ds.extTable.GetSelection()
		assert.Equal(t, 2, row)

		// Test Up at top
		ds.extTable.Select(0, 0)
		assert.NotNil(t, ds.inputCapture(eventUp))

		// Test Up with single ext group at row 0
		// We need a DirContext where the first group has 1 extension.
		// "Videos" (groupID: Video) might not be the first one if it's sorted.
		// SetDir sorts groups by title (if not Other).
		// "Videos" vs "Images".
		entriesSingle := []os.DirEntry{
			mockDirEntry{name: "video1.mp4", isDir: false},
		}
		ds.SetDir(&DirContext{Path: "/test", children: entriesSingle})
		// Rows:
		// 0: Videos (1 ext)
		// 1: .mp4
		ds.extTable.Select(1, 0)
		// Prev row is 0, which is "Videos" group.
		// It has 1 ext, and row == 1, so it should return nil.
		assert.Nil(t, ds.inputCapture(eventUp))

		// Test other key
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
		ds.SetDir(&DirContext{Path: "/test", children: entries})
		err := ds.GetSizes()
		assert.Error(t, err)
	})
}
