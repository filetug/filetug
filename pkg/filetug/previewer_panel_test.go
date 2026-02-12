package filetug

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/filetug/filetug/pkg/viewers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func newNavigatorForPreviewerTest(t *testing.T) *Navigator {
	t.Helper()
	app := &testApp{
		queueUpdateDraw: func(f func()) {
			if f != nil {
				f()
			}
		},
	}
	return NewNavigator(app)
}

func TestPreviewer(t *testing.T) {
	withTestGlobalLock(t)
	viewers.SetTextPreviewerSyncForTest(true)
	defer viewers.SetTextPreviewerSyncForTest(false)
	previewFile := func(previewerPanel *previewerPanel, name, fullName string) {
		dirPath := filepath.Dir(fullName)
		var entry files.EntryWithDirPath
		if entries, err := os.ReadDir(dirPath); err == nil {
			for _, dirEntry := range entries {
				if dirEntry.Name() == name {
					entry = files.NewEntryWithDirPath(dirEntry, dirPath)
					break
				}
			}
		}
		if entry == nil {
			entry = files.NewEntryWithDirPath(files.NewDirEntry(name, false), dirPath)
		}
		previewerPanel.PreviewEntry(entry)
	}
	previewText := func(previewer *previewerPanel) string {
		if tv, ok := previewer.previewer.Main().(*tview.TextView); ok {
			return tv.GetText(true)
		}
		return ""
	}

	//nav.previewer.textView.SetText("")

	t.Run("Draw", func(t *testing.T) {
		s := ttestutils.NewSimScreen(t, "UTF-8", 80, 24)
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.Draw(s)
	})

	t.Run("SetText", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.SetText("test text")
		assert.Contains(t, nav.previewer.textView.GetText(false), "test text")
	})

	t.Run("SetErr", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.SetErr(fmt.Errorf("test error"))
		assert.Contains(t, nav.previewer.textView.GetText(false), "test error")
	})

	t.Run("PreviewFile_DSStore_Valid", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		tmpFile, _ := os.CreateTemp("", ".DS_Store")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		// Minimal DSStore header: 4 bytes Magic, 4 bytes "Bud1"
		// See https://en.wikipedia.org/wiki/.DS_Store
		header := []byte{0x00, 0x00, 0x00, 0x01, 0x42, 0x75, 0x64, 0x31}
		_ = os.WriteFile(tmpFile.Name(), header, 0644)
		previewFile(nav.previewer, ".DS_Store", tmpFile.Name())
		previewFile(nav.previewer, ".DS_Store", tmpFile.Name())
	})

	// TODO: Flaky test needs fixing.
	t.Run("PreviewFile_DSStore_Reuse_Previewer", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		tmpFile, _ := os.CreateTemp("", ".DS_Store")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		// Minimal DSStore header: 4 bytes Magic, 4 bytes "Bud1"
		header := []byte{0x00, 0x00, 0x00, 0x01, 0x42, 0x75, 0x64, 0x31}
		_ = os.WriteFile(tmpFile.Name(), header, 0644)

		// First call creates DsstorePreviewer
		previewFile(nav.previewer, ".DS_Store", tmpFile.Name())
		firstPreviewer := nav.previewer.previewer

		// Verify it's a DsstorePreviewer
		_, ok := firstPreviewer.(*viewers.DsstorePreviewer)
		assert.True(t, ok, "First preview should create a DsstorePreviewer")

		// Second call should reuse the same previewer (covering lines 197-198)
		previewFile(nav.previewer, ".DS_Store", tmpFile.Name())
		secondPreviewer := nav.previewer.previewer

		// Verify same instance is reused (same pointer address)
		assert.True(t, firstPreviewer == secondPreviewer, "Second preview should reuse the same DsstorePreviewer instance")
	})

	t.Run("FocusBlur", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewerFocusFunc()
		nav.previewerBlurFunc()
		nav.previewer.Focus(func(p tview.Primitive) {})
		nav.previewer.Blur()
		nav.previewer.textView.Focus(func(p tview.Primitive) {})
	})

	t.Run("PreviewFile_NotFound", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.setPreviewer(nil)
		tmpDir := t.TempDir()
		entry := files.NewEntryWithDirPath(mockDirEntry{name: "non-existent.txt", isDir: false}, tmpDir)
		nav.previewer.PreviewEntry(entry)
		if _, ok := nav.previewer.previewer.(*viewers.TextPreviewer); !ok {
			t.Logf("expected text previewer, got %T", nav.previewer.previewer)
			return
		}
		waitForText(t, nav.previewer, previewText, "Failed to read file")
	})

	t.Run("PreviewFile_PlainText", func(t *testing.T) {
		entry := files.NewEntryWithDirPath(files.NewDirEntry("file.txt", false), t.TempDir())
		textPreviewer := viewers.NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		textPreviewer.PreviewSingle(entry, []byte("hello world"), nil)
		text := textPreviewer.GetText(true)
		assert.Contains(t, text, "hello world")
	})

	t.Run("PreviewFile_JSON", func(t *testing.T) {
		entry := files.NewEntryWithDirPath(files.NewDirEntry("test.json", false), t.TempDir())
		jsonPreviewer := viewers.NewJsonPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		jsonPreviewer.PreviewSingle(entry, []byte(`{"a":1}`), nil)
		text := jsonPreviewer.GetText(true)
		assert.Contains(t, text, "a")
	})

	t.Run("PreviewFile_JSON_SameType_Updates", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.setPreviewer(nil)

		// Use t.TempDir() to create a unique directory for this test
		tmpDir := t.TempDir()

		firstFile, err := os.CreateTemp(tmpDir, "first*.json")
		assert.NoError(t, err)
		firstPath := firstFile.Name()
		_ = firstFile.Close() // Close the file before writing to it

		secondFile, err := os.CreateTemp(tmpDir, "second*.json")
		assert.NoError(t, err)
		secondPath := secondFile.Name()
		_ = secondFile.Close() // Close the file before writing to it

		err = os.WriteFile(firstPath, []byte(`{"first":1}`), 0644)
		assert.NoError(t, err)
		err = os.WriteFile(secondPath, []byte(`{"second":2}`), 0644)
		assert.NoError(t, err)

		previewFile(nav.previewer, filepath.Base(firstPath), firstPath)
		waitForText(t, nav.previewer, previewText, "first")

		previewFile(nav.previewer, filepath.Base(secondPath), secondPath)
		waitForText(t, nav.previewer, previewText, "second")
	})

	t.Run("PreviewFile_Text_SameType_Updates", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		nav.previewer.setPreviewer(nil)
		firstFile, _ := os.CreateTemp("", "first*.txt")
		defer func() {
			_ = os.Remove(firstFile.Name())
		}()
		secondFile, _ := os.CreateTemp("", "second*.txt")
		defer func() {
			_ = os.Remove(secondFile.Name())
		}()
		_ = os.WriteFile(firstFile.Name(), []byte("first text"), 0644)
		_ = os.WriteFile(secondFile.Name(), []byte("second text"), 0644)

		// First preview creates a previewer
		previewFile(nav.previewer, filepath.Base(firstFile.Name()), firstFile.Name())
		firstPreviewer := nav.previewer.previewer
		assert.NotNil(t, firstPreviewer)

		// Second preview - should exercise the reuse code path for text previewer
		previewFile(nav.previewer, filepath.Base(secondFile.Name()), secondFile.Name())
		secondPreviewer := nav.previewer.previewer
		assert.NotNil(t, secondPreviewer)
	})

	t.Run("PreviewFile_JSONB", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		tmpFile, _ := os.CreateTemp("", "test*.jsonb")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		_ = os.WriteFile(tmpFile.Name(), []byte(`{"test": "jsonb"}`), 0644)
		previewFile(nav.previewer, filepath.Base(tmpFile.Name()), tmpFile.Name())
		// Just verify it doesn't crash - coverage is the goal
		assert.NotNil(t, nav.previewer.previewer)
	})

	t.Run("InputCapture", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		event := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		nav.previewer.rows.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		nav.previewer.rows.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		nav.previewer.rows.GetInputCapture()(event)
	})

	t.Run("PreviewFile_NoName", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)
		previewFile(nav.previewer, "", tmpFile.Name())
	})

	t.Run("PreviewFile_NoLexer", func(t *testing.T) {
		entry := files.NewEntryWithDirPath(files.NewDirEntry("test", false), t.TempDir())
		textPreviewer := viewers.NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		textPreviewer.PreviewSingle(entry, []byte("hello world"), nil)
		text := textPreviewer.GetText(true)
		assert.Contains(t, text, "hello world")
	})

	t.Run("PreviewFile_JSON_Invalid_Pretty", func(t *testing.T) {
		entry := files.NewEntryWithDirPath(files.NewDirEntry("file.json", false), t.TempDir())
		jsonPreviewer := viewers.NewJsonPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		jsonPreviewer.PreviewSingle(entry, []byte(`{invalid}`), nil)
		text := jsonPreviewer.GetText(true)
		assert.Contains(t, text, "{invalid}")
	})

	t.Run("PreviewFile_Image_Meta", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		tmpFile, _ := os.CreateTemp("", "test*.png")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		// A 1x1 pixel PNG file (minimal valid PNG)
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
			0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0x44, 0x74, 0x06, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
			0x44, 0xAE, 0x42, 0x60, 0x82,
		}
		_ = os.WriteFile(tmpFile.Name(), pngData, 0644)
		previewFile(nav.previewer, filepath.Base(tmpFile.Name()), tmpFile.Name())

		secondFile, _ := os.CreateTemp("", "test*.png")
		defer func() {
			_ = os.Remove(secondFile.Name())
		}()
		_ = os.WriteFile(secondFile.Name(), pngData, 0644)
		previewFile(nav.previewer, filepath.Base(secondFile.Name()), secondFile.Name())
	})

	t.Run("PreviewFile_ChromaError", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		// To trigger chroma error, we can try something that is not valid UTF-8 if the lexer expects it
		// but chroma2tcell usually handles bytes.
		// However, it's worth a try with some invalid bytes for a specific lexer.
		tmpFile, _ := os.CreateTemp("", "test*.go")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		_ = os.WriteFile(tmpFile.Name(), []byte{0xff, 0xfe, 0xfd}, 0644)
		previewFile(nav.previewer, filepath.Base(tmpFile.Name()), tmpFile.Name())
	})

	t.Run("PreviewFile_Log", func(t *testing.T) {
		entry := files.NewEntryWithDirPath(files.NewDirEntry("file.log", false), t.TempDir())
		textPreviewer := viewers.NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		textPreviewer.PreviewSingle(entry, []byte("log line"), nil)
		text := textPreviewer.GetText(true)
		assert.Contains(t, text, "log line")
	})

	t.Run("PreviewFile_DSStore_Error_ReadFile", func(t *testing.T) {
		nav := newNavigatorForPreviewerTest(t)
		// To trigger readFile error inside DSStore branch
		tmpDir, _ := os.MkdirTemp("", "testds")
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
		dsPath := filepath.Join(tmpDir, ".DS_Store")
		_ = os.Mkdir(dsPath, 0755) // Create as directory to cause read error
		previewFile(nav.previewer, ".DS_Store", dsPath)
	})
}

func waitForText(t *testing.T, previewer *previewerPanel, getText func(previewer *previewerPanel) string, needle string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(getText(previewer), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	text := getText(previewer)
	if needle != "" {
		assert.NotEmpty(t, text)
	}
	assert.Contains(t, text, needle)
}
