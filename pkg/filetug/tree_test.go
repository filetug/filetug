package filetug

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTree(t *testing.T) {
	nav := NewNavigator(nil)
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
		nav := NewNavigator(nil)
		tree := NewTree(nav)
		loading := tview.NewTreeNode(" Loading...")
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		queued := false
		done := make(chan struct{})
		var once sync.Once
		nav.queueUpdateDraw = func(f func()) {
			queued = true
			f()
			tree.rootNode.ClearChildren()
			once.Do(func() {
				close(done)
			})
		}

		go tree.doLoadingAnimation(loading)
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for queueUpdateDraw")
		}
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

		nodeContext := files.NewDirContext(nil, "/test", nil)
		node := tview.NewTreeNode("test").SetReference(nodeContext)
		tree.changed(node)
	})

	t.Run("setError", func(t *testing.T) {
		nodeContext := files.NewDirContext(nil, "/test", nil)
		node := tview.NewTreeNode("test").SetReference(nodeContext)
		tree.setError(node, fmt.Errorf("test error"))
	})

	t.Run("focus_blur", func(t *testing.T) {
		tree.focus()
		tree.blur()
	})

	t.Run("inputCapture", func(t *testing.T) {
		root := tree.tv.GetRoot()
		rootContext := files.NewDirContext(nil, "/test", nil)
		root.SetReference(rootContext)
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

	t.Run("inputCapture_KeyLeft_UnknownRef", func(t *testing.T) {
		badNode := tview.NewTreeNode("bad")
		badNode.SetReference("bad")
		tree.tv.SetCurrentNode(badNode)
		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		res := tree.inputCapture(eventLeft)
		assert.Equal(t, eventLeft, res)
	})

	t.Run("inputCapture_KeyEnter_UnknownRef", func(t *testing.T) {
		badNode := tview.NewTreeNode("bad")
		badNode.SetReference("bad")
		tree.tv.SetCurrentNode(badNode)
		eventEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := tree.inputCapture(eventEnter)
		assert.Equal(t, eventEnter, res)
	})

	t.Run("inputCapture_KeyRune_GlobalHandled", func(t *testing.T) {
		store := newMockStoreWithRootTitle(t, url.URL{Scheme: "mock", Path: "/"}, "Root")
		store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		tree.nav.store = store
		tree.nav.queueUpdateDraw = func(f func()) {
			f()
		}
		eventRune := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
		res := tree.inputCapture(eventRune)
		assert.Nil(t, res)
	})

	t.Run("inputCapture_KeyRune_SpaceIgnored", func(t *testing.T) {
		tree.searchPattern = ""
		eventRune := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
		res := tree.inputCapture(eventRune)
		assert.Equal(t, eventRune, res)
		assert.Equal(t, "", tree.searchPattern)
	})

	t.Run("SetSearch", func(t *testing.T) {
		tree.SetSearch("test")
	})

	t.Run("highlightTreeNodes_skipsRoot", func(t *testing.T) {
		rootContext := files.NewDirContext(nil, "/Users/demo", nil)
		childContext := files.NewDirContext(nil, "/Users/demo/alpha", nil)
		root := tview.NewTreeNode("..").SetReference(rootContext)
		child := tview.NewTreeNode("alpha").SetReference(childContext)
		root.AddChild(child)

		searchCtx := &searchContext{pattern: "al"}
		highlightTreeNodes(root, searchCtx, true)

		assert.Equal(t, "..", root.GetText())
		assert.Equal(t, "[black:lightgreen]al[-:-]pha", child.GetText())
		assert.Equal(t, child, searchCtx.firstPrefixed)
		assert.Len(t, searchCtx.found, 1)
	})

	t.Run("highlightTreeNodes_rootMatchIgnored", func(t *testing.T) {
		rootContext := files.NewDirContext(nil, "/alpha", nil)
		root := tview.NewTreeNode("..").SetReference(rootContext)
		searchCtx := &searchContext{pattern: "al"}
		highlightTreeNodes(root, searchCtx, true)

		assert.Equal(t, "..", root.GetText())
		assert.Nil(t, searchCtx.firstPrefixed)
		assert.Nil(t, searchCtx.firstContains)
		assert.Len(t, searchCtx.found, 0)
	})

	t.Run("setCurrentDir", func(t *testing.T) {
		dirContext := files.NewDirContext(tree.nav.store, "/", nil)
		tree.setCurrentDir(dirContext)
	})

	t.Run("setCurrentDir_Nil", func(t *testing.T) {
		tree.setCurrentDir(nil)
	})

	t.Run("setDirContext", func(t *testing.T) {
		root := tree.tv.GetRoot()
		dc := files.NewDirContext(nil, "/test", []os.DirEntry{
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
		})
		tree.setDirContext(context.Background(), root, dc)
	})

	t.Run("setError", func(t *testing.T) {
		root := tree.tv.GetRoot()
		tree.setError(root, context.DeadlineExceeded)
	})

	t.Run("getNodePath", func(t *testing.T) {
		emptyPath := getNodePath(nil)
		assert.Equal(t, "", emptyPath)

		root := tree.tv.GetRoot()
		rootContext := files.NewDirContext(tree.nav.store, "/", nil)
		root.SetReference(rootContext)
		child := tview.NewTreeNode("child")
		childContext := files.NewDirContext(tree.nav.store, "/child", nil)
		child.SetReference(childContext)
		root.AddChild(child)
		childPath := getNodePath(child)
		assert.Equal(t, "/child", childPath)

		badRefNode := tview.NewTreeNode("bad")
		badRefNode.SetReference("bad")
		badPath := getNodePath(badRefNode)
		assert.Equal(t, "", badPath)
	})

	t.Run("GetCurrentEntry_NonDirContext", func(t *testing.T) {
		node := tview.NewTreeNode("bad")
		node.SetReference("bad")
		tree.tv.SetCurrentNode(node)
		entry := tree.GetCurrentEntry()
		assert.Nil(t, entry)
	})
}
