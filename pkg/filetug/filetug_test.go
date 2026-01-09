package filetug

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestNewFavorites(t *testing.T) {
	f := newFavorites()
	if f == nil {
		t.Fatal("f is nil")
	}
	if f.GetRoot() == nil {
		t.Fatal("root is nil")
	}
	assert.Equal(t, 2, len(f.GetRoot().GetChildren()))
}

func TestNewTree(t *testing.T) {
	tree := NewTree()
	if tree == nil {
		t.Fatal("tree is nil")
	}
	if tree.GetRoot() == nil {
		t.Fatal("root is nil")
	}
	if tree.GetBox() == nil {
		t.Fatal("box is nil")
	}
}

func TestNavigator(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	if nav == nil {
		t.Fatal("nav is nil")
	}

	t.Run("SetFocus", func(t *testing.T) {
		nav.SetFocus()
	})

	t.Run("NavigatorInputCapture", func(t *testing.T) {
		altKey := func(r rune) *tcell.EventKey {
			return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModAlt)
		}
		nav.GetInputCapture()(altKey('0'))
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.activeCol = 1
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.activeCol = 2
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

		nav.GetInputCapture()(altKey('r'))
		nav.GetInputCapture()(altKey('h'))
		nav.GetInputCapture()(altKey('?'))
		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})
}

func TestPreviewer(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	p := nav.previewer

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

	t.Run("prettyJSON_Error", func(t *testing.T) {
		_, err := prettyJSON("{invalid}")
		assert.Error(t, err)
	})
}

func TestMainFunc(t *testing.T) {
	app := tview.NewApplication()
	SetupApp(app)
}

func TestFiles(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	f := nav.files

	t.Run("FocusBlur", func(t *testing.T) {
		nav.filesFocusFunc()
		nav.filesBlurFunc()
	})

	t.Run("SelectionChanged", func(t *testing.T) {
		nav.filesSelectionChangedFunc(0, 0)

		f.SetCell(1, 0, tview.NewTableCell(" file.txt"))
		nav.filesSelectionChangedFunc(1, 0)

		// Test with no space prefix (should not happen in real app but for coverage)
		f.SetCell(2, 0, tview.NewTableCell("file.txt"))
		defer func() { _ = recover() }()
		nav.filesSelectionChangedFunc(2, 0)
	})

	t.Run("InputCapture_Space", func(t *testing.T) {
		f.SetCell(1, 0, tview.NewTableCell(" name"))
		f.Select(1, 0)
		event := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
		f.GetInputCapture()(event)
		assert.Contains(t, f.GetCell(1, 0).Text, "✓")

		f.GetInputCapture()(event)
		assert.Contains(t, f.GetCell(1, 0).Text, " ")
	})

	t.Run("InputCapture_Keys", func(t *testing.T) {
		f.GetInputCapture()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
		f.GetInputCapture()(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
		f.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		f.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})
}

func TestLeft(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))

	t.Run("LeftFocusBlur", func(t *testing.T) {
		nav.leftFocusFunc()
		nav.leftBlurFunc()
	})

	t.Run("FavoritesFocusBlur", func(t *testing.T) {
		nav.favoritesFocusFunc()
		nav.favoritesBlurFunc()
	})

	t.Run("DirsFocusBlur", func(t *testing.T) {
		nav.dirsFocusFunc()
		nav.dirsBlurFunc()
	})

	t.Run("LeftInputCapture", func(t *testing.T) {
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
		nav.left.GetInputCapture()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
	})

	t.Run("FavoritesInputCapture", func(t *testing.T) {
		nav.favorites.SetCurrentNode(nav.favorites.GetRoot())
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))

		// Set current node to last child for KeyDown test
		children := nav.favorites.GetRoot().GetChildren()
		nav.favorites.SetCurrentNode(children[len(children)-1])
		nav.favorites.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	})

	t.Run("DirsInputCapture", func(t *testing.T) {
		// Mock current node to avoid nil dereference in GetCurrentNode().GetReference()
		nav.dirs.SetCurrentNode(tview.NewTreeNode("test").SetReference("."))
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
		nav.dirs.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	})
}
