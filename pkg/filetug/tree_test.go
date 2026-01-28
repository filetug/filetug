package filetug

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
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
			f()
			tree.rootNode.ClearChildren()
		}

		go func() {
			tree.doLoadingAnimation(loading)
		}()
		time.Sleep(110 * time.Millisecond)
		assert.GreaterOrEqual(t, drawUpdatesCount, 1)
		// Since we cleared children in a goroutine, it should have iterated a few times then stopped.
	})

	t.Run("doLoadingAnimation_queueUpdateDrawExecutes", func(t *testing.T) {
		nav := NewNavigator(app)
		tree := NewTree(nav)
		loading := tview.NewTreeNode(" Loading...")
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		queued := false
		nav.queueUpdateDraw = func(f func()) {
			queued = true
			f()
			tree.rootNode.ClearChildren()
		}

		tree.doLoadingAnimation(loading)
		assert.True(t, queued)
	})

	t.Run("doLoadingAnimation_withoutQueueUpdateDraw", func(t *testing.T) {
		loading := tview.NewTreeNode(" Loading...")
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		originalNav := tree.nav
		tree.nav = nil
		defer func() {
			tree.nav = originalNav
		}()

		done := make(chan struct{})
		go func() {
			tree.doLoadingAnimation(loading)
			close(done)
		}()

		updatedText := ""
		deadline := time.Now().Add(300 * time.Millisecond)
		for time.Now().Before(deadline) {
			text := loading.GetText()
			if text != " Loading..." {
				updatedText = text
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		tree.rootNode.ClearChildren()

		timeout := time.After(300 * time.Millisecond)
		select {
		case <-done:
		case <-timeout:
			t.Fatalf("loading animation did not stop")
		}

		if updatedText == "" {
			t.Fatalf("expected loading text to update")
		}
	})

	t.Run("changed", func(t *testing.T) {
		root := tree.tv.GetRoot()
		tree.changed(root)

		// Test with string reference
		node := tview.NewTreeNode("test").SetReference("/test")
		tree.changed(node)
	})

	t.Run("changed_updatesDirSummaryPreview", func(t *testing.T) {
		nav := NewNavigator(app)
		nav.store = osfile.NewStore("/")
		nav.dirSummary = newTestDirSummary(nav)
		tree := NewTree(nav)

		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "alpha.txt"), []byte("data"), 0644)
		assert.NoError(t, err)

		done := make(chan struct{})
		var once sync.Once
		nav.queueUpdateDraw = func(f func()) {
			f()
			once.Do(func() {
				close(done)
			})
		}

		node := tview.NewTreeNode("temp").SetReference(tempDir)
		tree.changed(node)

		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for dir summary update")
		}

		if assert.Len(t, nav.dirSummary.ExtStats, 1) {
			assert.Equal(t, ".txt", nav.dirSummary.ExtStats[0].ID)
		}
	})

	t.Run("updateDirSummaryPreview_usesNodeReference", func(t *testing.T) {
		nav := NewNavigator(app)
		nav.dirSummary = newTestDirSummary(nav)
		nav.store = osfile.NewStore("/")
		tree := NewTree(nav)

		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "alpha.txt"), []byte("data"), 0644)
		assert.NoError(t, err)

		nav.current.dir = t.TempDir()
		node := tview.NewTreeNode("temp").SetReference(tempDir)

		tree.updateDirSummaryPreview(node)

		if assert.Len(t, nav.dirSummary.ExtStats, 1) {
			assert.Equal(t, ".txt", nav.dirSummary.ExtStats[0].ID)
		}
	})

	t.Run("updateDirSummaryPreview_earlyReturn", func(t *testing.T) {
		nav := NewNavigator(app)
		tree := NewTree(nav)
		tree.updateDirSummaryPreview(nil)
	})

	t.Run("updateDirSummaryPreview_emptyRef", func(t *testing.T) {
		nav := NewNavigator(app)
		nav.dirSummary = newTestDirSummary(nav)
		tree := NewTree(nav)
		node := tview.NewTreeNode("empty").SetReference("")
		tree.updateDirSummaryPreview(node)
	})

	t.Run("updateDirSummaryPreview_nonStringRef", func(t *testing.T) {
		nav := NewNavigator(app)
		nav.dirSummary = newTestDirSummary(nav)
		tree := NewTree(nav)
		node := tview.NewTreeNode("bad").SetReference(123)
		tree.updateDirSummaryPreview(node)
	})

	t.Run("updateDirSummaryPreview_storeNil", func(t *testing.T) {
		oldGetState := getState
		getState = func() (*ftstate.State, error) {
			return nil, errors.New("disabled")
		}
		defer func() {
			getState = oldGetState
		}()
		nav := NewNavigator(app)
		nav.dirSummary = newTestDirSummary(nav)
		nav.store = nil
		tree := NewTree(nav)
		node := tview.NewTreeNode("temp").SetReference(t.TempDir())
		tree.updateDirSummaryPreview(node)
	})

	t.Run("updateDirSummaryPreview_readDirError", func(t *testing.T) {
		nav := NewNavigator(app)
		nav.dirSummary = newTestDirSummary(nav)
		nav.store = &mockStoreWithHooks{readDirErr: errors.New("read failed")}
		tree := NewTree(nav)
		node := tview.NewTreeNode("temp").SetReference(t.TempDir())
		tree.updateDirSummaryPreview(node)
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

	t.Run("highlightTreeNodes_skipsRoot", func(t *testing.T) {
		root := tview.NewTreeNode("..").SetReference("/Users/demo")
		child := tview.NewTreeNode("alpha").SetReference("/Users/demo/alpha")
		root.AddChild(child)

		searchCtx := &searchContext{pattern: "al"}
		highlightTreeNodes(root, searchCtx, true)

		assert.Equal(t, "..", root.GetText())
		assert.Equal(t, "[black:lightgreen]al[-:-]pha", child.GetText())
		assert.Equal(t, child, searchCtx.firstPrefixed)
		assert.Len(t, searchCtx.found, 1)
	})

	t.Run("highlightTreeNodes_rootMatchIgnored", func(t *testing.T) {
		root := tview.NewTreeNode("..").SetReference("/alpha")
		searchCtx := &searchContext{pattern: "al"}
		highlightTreeNodes(root, searchCtx, true)

		assert.Equal(t, "..", root.GetText())
		assert.Nil(t, searchCtx.firstPrefixed)
		assert.Nil(t, searchCtx.firstContains)
		assert.Len(t, searchCtx.found, 0)
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
