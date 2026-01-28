package filetug

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/rivo/tview"
)

func TestNavigator_Delete_And_Operations(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

	// Setup a temporary file to delete
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testdelete.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	assert.NoError(t, err)

	nav.store = osfile.NewStore(tmpDir)

	t.Run("delete_no_selection", func(t *testing.T) {
		// Mock getCurrentBrowser to return something by focusing files
		nav.files.Focus(func(p tview.Primitive) {})
		nav.activeCol = 1

		nav.files.rows = &FileRows{} // No entries

		nav.delete()
	})

	t.Run("delete_with_selection", func(t *testing.T) {
		entries, err := nav.store.ReadDir(context.Background(), tmpDir)
		assert.NoError(t, err)

		nav.activeCol = 1

		// Use real DirContext to avoid nil dereference in GetCurrentEntry
		rows := NewFileRows(&DirContext{Path: tmpDir, Store: nav.store, children: entries})
		nav.files.SetRows(rows, false)
		nav.files.Focus(func(p tview.Primitive) {})

		// Ensure we selected something
		nav.files.table.Select(0, 0)

		// Call delete
		nav.delete()

		// Wait for operation to complete (it's async)
		time.Sleep(200 * time.Millisecond)

		// Check if file is gone
		_, err = os.Stat(tmpFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("getCurrentBrowser", func(t *testing.T) {
		// Test files focus
		nav.files.Focus(func(p tview.Primitive) {})
		nav.activeCol = 1
		assert.True(t, nav.files == nav.getCurrentBrowser())

		// Test tree focus
		nav.dirsTree.Focus(func(p tview.Primitive) {})
		nav.activeCol = 0
		assert.True(t, nav.dirsTree == nav.getCurrentBrowser())
	})
}

func TestFilesPanel_GetCurrentEntry_EdgeCases(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	fp := newFiles(nav)

	t.Run("empty_rows", func(t *testing.T) {
		fp.rows = &FileRows{}
		assert.Equal(t, (files.EntryWithDirPath)(nil), fp.GetCurrentEntry())
	})

	t.Run("entry_with_no_dir_path_but_rows_has_dir", func(t *testing.T) {
		mEntry := mockDirEntry{name: "test.txt"}
		rows := &FileRows{
			VisibleEntries: []files.EntryWithDirPath{
				{DirEntry: mEntry},
			},
			Dir: &DirContext{Path: "/some/path"},
		}
		fp.rows = rows
		fp.table.Select(0, 0)

		entry := fp.GetCurrentEntry()
		assert.True(t, entry != nil)
		assert.Equal(t, "/some/path", entry.Dir)
		assert.Equal(t, "test.txt", entry.Name())
	})
}

func TestOperation_Coverage(t *testing.T) {
	t.Run("NewOperation", func(t *testing.T) {
		done := make(chan bool)
		_ = NewOperation("test", func(ctx context.Context, reportProgress ProgressReporter) error {
			reportProgress(OperationProgress{Total: 1, Done: 1})
			done <- true
			return nil
		}, func(progress OperationProgress) {
			assert.Equal(t, 1, progress.Total)
		})

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	})
}
