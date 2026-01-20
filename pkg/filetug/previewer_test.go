package filetug

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/datatug/filetug/pkg/sneatv/ttestutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestPreviewer(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	p := nav.previewer

	t.Run("Draw", func(t *testing.T) {
		s := ttestutils.NewSimScreen(t, "UTF-8", 80, 24)
		p.Draw(s)
	})

	t.Run("FocusBlur", func(t *testing.T) {
		nav.previewerFocusFunc()
		nav.previewerBlurFunc()
	})

	t.Run("TextViewFocus", func(t *testing.T) {
		// p.textView.GetFocusFunc()() // Not available
	})

	t.Run("PreviewFile_NotFound", func(t *testing.T) {
		p.PreviewFile("non-existent.txt", "non-existent.txt")
		assert.Contains(t, p.textView.GetText(false), "Error reading file")
	})

	t.Run("PreviewFile_PlainText", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)

		p.PreviewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		assert.Contains(t, p.textView.GetText(false), "hello world")
	})

	t.Run("PreviewFile_JSON", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.json")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte(`{"a":1}`), 0644)
		assert.NoError(t, err)

		p.PreviewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		// Colorized output will have tags, but GetText(false) should strip them or show them depending on dynamic colors
		// tview.TextView.GetText(false) returns the text without tags if dynamic colors are enabled.
		assert.Contains(t, p.textView.GetText(false), "a")
	})

	t.Run("InputCapture", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		p.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		p.GetInputCapture()(event)

		event = tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		p.GetInputCapture()(event)
	})

	t.Run("PreviewFile_NoName", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)
		p.PreviewFile("", tmpFile.Name())
	})

	t.Run("PreviewFile_NoLexer", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("hello world"), 0644)
		assert.NoError(t, err)

		p.PreviewFile("noext", tmpFile.Name())
		assert.Contains(t, p.textView.GetText(false), "hello world")
	})

	t.Run("PreviewFile_JSON_Invalid_Pretty", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.json")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte(`{invalid}`), 0644)
		assert.NoError(t, err)

		p.PreviewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		assert.Contains(t, p.textView.GetText(true), "{invalid}")
	})

	t.Run("prettyJSON_Error", func(t *testing.T) {
		_, err := prettyJSON("{invalid}")
		assert.Error(t, err)
	})

	t.Run("PreviewFile_Image", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.png")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		// Just an empty file might not work if imageviewer checks for headers, but let's see
		err := os.WriteFile(tmpFile.Name(), []byte("not an image but .png extension"), 0644)
		assert.NoError(t, err)

		p.PreviewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
	})

	t.Run("PreviewFile_Log", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.log")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		err := os.WriteFile(tmpFile.Name(), []byte("log line"), 0644)
		assert.NoError(t, err)

		p.PreviewFile(filepath.Base(tmpFile.Name()), tmpFile.Name())
		assert.Contains(t, p.textView.GetText(false), "log line")
	})

	t.Run("PreviewFile_DSStore", func(t *testing.T) {
		// Mocking a .DS_Store is hard, let's just try to call it with an empty file
		tmpFile, _ := os.CreateTemp("", ".DS_Store")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		_ = os.WriteFile(tmpFile.Name(), []byte("invalid ds_store"), 0644)

		p.PreviewFile(".DS_Store", tmpFile.Name())
		assert.Contains(t, p.textView.GetText(false), "invalid file header")
	})

	t.Run("readFile", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "test*.txt")
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		content := "0123456789"
		err := os.WriteFile(tmpFile.Name(), []byte(content), 0644)
		assert.NoError(t, err)

		t.Run("max_0", func(t *testing.T) {
			data, err := p.readFile(tmpFile.Name(), 0)
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		})

		t.Run("max_5", func(t *testing.T) {
			data, err := p.readFile(tmpFile.Name(), 5)
			assert.NoError(t, err)
			assert.Equal(t, "01234", string(data))
		})

		t.Run("max_minus_5", func(t *testing.T) {
			data, err := p.readFile(tmpFile.Name(), -5)
			assert.NoError(t, err)
			assert.Equal(t, "56789", string(data))
		})

		t.Run("max_minus_20", func(t *testing.T) {
			data, err := p.readFile(tmpFile.Name(), -20)
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		})
	})
}
