package filetug

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestTree(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tree := NewTree(nav)

	t.Run("onStoreChange", func(t *testing.T) {
		tree.onStoreChange()
	})

	t.Run("Draw", func(t *testing.T) {
		screen := tcell.NewSimulationScreen("")
		_ = screen.Init()
		tree.Draw(screen)

		// Test Draw with suffix space
		root := tree.tv.GetRoot()
		root.SetText("root ")
		tree.Draw(screen)
	})

	t.Run("doLoadingAnimation", func(t *testing.T) {
		loading := tview.NewTreeNode(" Loading...")
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		// Mock app and queue
		app := tview.NewApplication()
		tree.nav.app = app

		// We need to avoid infinite recursion and hangs.
		// We want to test at least one iteration.
		// We can use a channel to signal when SetText is called, but SetText doesn't have a callback.
		// However, we can check if the text changed after a short delay.

		drawUpdatesCount := 0
		oldQueueUpdateDraw := nav.queueUpdateDraw
		defer func() {
			nav.queueUpdateDraw = oldQueueUpdateDraw
		}()
		tree.nav.queueUpdateDraw = func(f func()) {
			drawUpdatesCount++
		}

		go func() {
			tree.doLoadingAnimation(loading)
		}()
		time.Sleep(110 * time.Millisecond)
		assert.GreaterOrEqual(t, drawUpdatesCount, 1)
		// Since we cleared children in a goroutine, it should have iterated a few times then stopped.
	})

	t.Run("changed", func(t *testing.T) {
		root := tree.tv.GetRoot()
		tree.changed(root)

		// Test with string reference
		node := tview.NewTreeNode("test").SetReference("/test")
		tree.changed(node)
	})

	t.Run("setError", func(t *testing.T) {
		node := tview.NewTreeNode("test").SetReference("/test")
		tree.setError(node, fmt.Errorf("test error"))
	})

	t.Run("focus_blur", func(t *testing.T) {
		tree.focus()
		tree.blur()
	})

	t.Run("inputCapture", func(t *testing.T) {
		root := tree.tv.GetRoot()
		root.SetReference("/test")
		tree.tv.SetCurrentNode(root)

		// Test Right
		eventRight := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
		tree.inputCapture(eventRight)

		// Test Left
		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		tree.inputCapture(eventLeft)

		// Test Enter
		eventEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		tree.inputCapture(eventEnter)

		// Test Up
		eventUp := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		tree.inputCapture(eventUp)

		// Test Backspace
		tree.searchPattern = "abc"
		eventBS := tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)
		tree.inputCapture(eventBS)

		// Test Escape
		eventEsc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
		tree.inputCapture(eventEsc)

		// Test Rune
		eventRune := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
		tree.inputCapture(eventRune)
	})

	t.Run("SetSearch", func(t *testing.T) {
		tree.SetSearch("test")
	})

	t.Run("setCurrentDir", func(t *testing.T) {
		tree.setCurrentDir("/")
	})

	t.Run("setDirContext", func(t *testing.T) {
		root := tree.tv.GetRoot()
		dc := &DirContext{
			Path: "/test",
			children: []os.DirEntry{
				mockDirEntry{name: "dir1", isDir: true},
				mockDirEntry{name: "file1", isDir: false},
				mockDirEntry{name: ".hidden", isDir: true},
				mockDirEntry{name: "Library", isDir: true},
				mockDirEntry{name: "Users", isDir: true},
				mockDirEntry{name: "Applications", isDir: true},
				mockDirEntry{name: "Music", isDir: true},
				mockDirEntry{name: "Movies", isDir: true},
				mockDirEntry{name: "Pictures", isDir: true},
				mockDirEntry{name: "Desktop", isDir: true},
				mockDirEntry{name: "DataTug", isDir: true},
				mockDirEntry{name: "Documents", isDir: true},
				mockDirEntry{name: "Public", isDir: true},
				mockDirEntry{name: "Temp", isDir: true},
				mockDirEntry{name: "System", isDir: true},
				mockDirEntry{name: "bin", isDir: true},
				mockDirEntry{name: "private", isDir: true},
			},
		}
		tree.setDirContext(context.Background(), root, dc)
	})

	t.Run("setError", func(t *testing.T) {
		root := tree.tv.GetRoot()
		tree.setError(root, context.DeadlineExceeded)
	})

	t.Run("getNodePath", func(t *testing.T) {
		root := tree.tv.GetRoot()
		root.SetReference("/")
		child := tview.NewTreeNode("child")
		child.SetReference("/child")
		root.AddChild(child)
		getNodePath(child)
	})
}
