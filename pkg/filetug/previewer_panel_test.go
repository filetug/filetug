package filetug

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func waitForText(t *testing.T, getText func() string, needle string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if strings.Contains(getText(), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Contains(t, getText(), needle)
}

func TestPreviewer(t *testing.T) {
	//t.Parallel()
	oldGetState := getState
	getState = func() (*ftstate.State, error) {
		return nil, errors.New("disabled")
	}
	defer func() {
		getState = oldGetState
	}()

	nav, _, _ := newNavigatorForTest(t)
	if nav == nil {
		t.Fatal("navigator is nil")
	}

	p := nav.previewer

	previewFile := func(name, fullName string) {
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
		p.PreviewEntry(entry)
	}
	previewText := func() string {
		if tv, ok := p.previewer.Main().(*tview.TextView); ok {
			return tv.GetText(true)
		}
		return ""
	}

	p.textView.SetText("")

	t.Run("Draw", func(t *testing.T) {
		s := ttestutils.NewSimScreen(t, "UTF-8", 80, 24)
		p.Draw(s)
	})

	t.Run("SetText", func(t *testing.T) {
		p.SetText("test text")
		assert.Contains(t, p.textView.GetText(false), "test text")
	})

	t.Run("SetErr", func(t *testing.T) {
		p.SetErr(fmt.Errorf("test error"))
		assert.Contains(t, p.textView.GetText(false), "test error")
	})

	t.Run("PreviewFile_DSStore_Valid", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", ".DS_Store")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		// Minimal DSStore header: 4 bytes Magic, 4 bytes "Bud1"
		// See https://en.wikipedia.org/wiki/.DS_Store
		header := []byte{0x00, 0x00, 0x00, 0x01, 0x42, 0x75, 0x64, 0x31}
		_ = os.WriteFile(tmpFile.Name(), header, 0644)
		previewFile(".DS_Store", tmpFile.Name())
		previewFile(".DS_Store", tmpFile.Name())
	})

	t.Run("FocusBlur", func(t *testing.T) {
		nav.previewerFocusFunc()
		nav.previewerBlurFunc()
		p.Focus(func(p tview.Primitive) {})
		p.Blur()
		p.textView.Focus(func(p tview.Primitive) {})
	})

	t.Run("PreviewFile_NotFound", func(t *testing.T) {
		previewFile("non-existent.txt", "non-existent.txt")
		waitForText(t, previewText, "Failed to read file")
	})

	t.Run("PreviewFile_PlainText", func(t *testing.T) {
		p.setPreviewer(nil)
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		waitForText(t, previewText, "hello world")
	})

	t.Run("PreviewFile_JSON", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.json")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte(`{"a":1}`), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		// Colorized output will have tags, but GetText(false) should strip them or show them depending on dynamic colors
		// tview.TextView.GetText(false) returns the text without tags if dynamic colors are enabled.
		waitForText(t, previewText, "a")
	})

	t.Run("PreviewFile_JSON_SameType_Updates", func(t *testing.T) {
		p.setPreviewer(nil)
		firstFile, _ := os.CreateTemp("", "first*.json")
		defer func() {
			_ = os.Remove(firstFile.Name())
		}()
		secondFile, _ := os.CreateTemp("", "second*.json")
		defer func() {
			_ = os.Remove(secondFile.Name())
		}()
		err := os.WriteFile(firstFile.Name(), []byte(`{"first":1}`), 0644)
		assert.NoError(t, err)
		err = os.WriteFile(secondFile.Name(), []byte(`{"second":2}`), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(firstFile.Name()), firstFile.Name())
		waitForText(t, previewText, "first")

		previewFile(filepath.Base(secondFile.Name()), secondFile.Name())
		waitForText(t, previewText, "second")
		assert.NotContains(t, previewText(), "first")
	})

	t.Run("InputCapture", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		p.rows.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		p.rows.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		p.rows.GetInputCapture()(event)
	})

	t.Run("PreviewFile_NoName", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)
		previewFile("", tmpFile.Name())
	})

	t.Run("PreviewFile_NoLexer", func(t *testing.T) {
		p.setPreviewer(nil)
		tmpFile, _ := os.CreateTemp("", "test")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		waitForText(t, previewText, "hello world")
	})

	t.Run("PreviewFile_JSON_Invalid_Pretty", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.json")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte(`{invalid}`), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		waitForText(t, previewText, "{invalid}")
	})

	t.Run("PreviewFile_Image_Meta", func(t *testing.T) {
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
		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())

		secondFile, _ := os.CreateTemp("", "test*.png")
		defer func() {
			_ = os.Remove(secondFile.Name())
		}()
		_ = os.WriteFile(secondFile.Name(), pngData, 0644)
		previewFile(filepath.Base(secondFile.Name()), secondFile.Name())
	})

	t.Run("PreviewFile_ChromaError", func(t *testing.T) {
		// To trigger chroma error, we can try something that is not valid UTF-8 if the lexer expects it
		// but chroma2tcell usually handles bytes.
		// However, it's worth a try with some invalid bytes for a specific lexer.
		tmpFile, _ := os.CreateTemp("", "test*.go")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		_ = os.WriteFile(tmpFile.Name(), []byte{0xff, 0xfe, 0xfd}, 0644)
		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
	})

	t.Run("PreviewFile_Log", func(t *testing.T) {
		p.setPreviewer(nil)
		tmpFile, _ := os.CreateTemp("", "test*.log")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("log line"), 0644)
		assert.NoError(t, err)

		previewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		waitForText(t, previewText, "log line")
	})

	t.Run("PreviewFile_DSStore_Error_ReadFile", func(t *testing.T) {
		// To trigger readFile error inside DSStore branch
		tmpDir, _ := os.MkdirTemp("", "testds")
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
		dsPath := filepath.Join(tmpDir, ".DS_Store")
		_ = os.Mkdir(dsPath, 0755) // Create as directory to cause read error
		previewFile(".DS_Store", dsPath)
	})
}
