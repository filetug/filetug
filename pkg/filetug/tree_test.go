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
	"github.com/filetug/filetug/pkg/tviewmocks"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTree(t *testing.T) {
	t.Parallel()

	t.Run("onStoreChange", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.onStoreChange()
	})

	t.Run("Draw", func(t *testing.T) {
		t.Parallel()
		screen := tcell.NewSimulationScreen("")
		err := screen.Init()
		assert.NoError(t, err, "Failed to initialize simulation screen")

		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.Draw(screen)

		// Test Draw with suffix space
		root := tree.tv.GetRoot()
		root.SetText("root ")
		tree.Draw(screen)
	})

	t.Run("doLoadingAnimation", func(t *testing.T) {
		//t.Parallel()
		loading := tview.NewTreeNode(" Loading...")
		nav, app, _ := newNavigatorForTest(t)
		tree := NewTree(nav) //tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		done := make(chan struct{})
		var texts []string
		var updatesCount int
		app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes().DoAndReturn(func(f func()) {
			if f != nil {
				f()
			}
			updatesCount++
			if updatesCount >= 4 {
				select {
				case done <- struct{}{}:
				default:
				}
				return
			}
			texts = append(texts, loading.GetText())
			if updatesCount >= 3 {
				tree.rootNode.ClearChildren()
				tree.rootNode.AddChild(tview.NewTreeNode("sub_dir_1"))
			}
		})

		go func() {
			tree.doLoadingAnimation(loading)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for queueUpdateDraw")
		}
		assert.Greater(t, len(texts), 0)
	})

	t.Run("doLoadingAnimation_queueUpdateDrawExecutes", func(t *testing.T) {
		t.Parallel()
		nav, app, ctrl := newNavigatorForTest(t)
		ctrl.Finish()
		app = tviewmocks.NewMockApp(ctrl)
		nav.app = app
		tree := NewTree(nav)
		loading := tview.NewTreeNode(" Loading...")
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

		queued := false
		done := make(chan struct{})
		var once sync.Once
		app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes().DoAndReturn(func(f func()) {
			queued = true
			if f != nil {
				f()
			}
			tree.rootNode.ClearChildren()
			once.Do(func() {
				close(done)
			})
		})

		go tree.doLoadingAnimation(loading)
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for queueUpdateDraw")
		}
		assert.True(t, queued)
	})

	t.Run("doLoadingAnimation_withoutQueueUpdateDraw", func(t *testing.T) {
		t.Parallel()
		loading := tview.NewTreeNode(" Loading...")
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.rootNode.ClearChildren()
		tree.rootNode.AddChild(loading)

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
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		root := tree.tv.GetRoot()
		tree.changed(root)

		nodeContext := files.NewDirContext(nil, "/test", nil)
		node := tview.NewTreeNode("test").SetReference(nodeContext)
		tree.changed(node)
	})

	t.Run("setError", func(t *testing.T) {
		t.Parallel()
		nodeContext := files.NewDirContext(nil, "/test", nil)
		node := tview.NewTreeNode("test").SetReference(nodeContext)
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.setError(node, fmt.Errorf("test error"))
	})

	t.Run("focus_blur", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.focus()
		tree.blur()
	})

	t.Run("inputCapture", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
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
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		badNode := tview.NewTreeNode("bad")
		badNode.SetReference("bad")
		tree.tv.SetCurrentNode(badNode)
		eventLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
		res := tree.inputCapture(eventLeft)
		assert.Equal(t, eventLeft, res)
	})

	t.Run("inputCapture_KeyEnter_UnknownRef", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		badNode := tview.NewTreeNode("bad")
		badNode.SetReference("bad")
		tree.tv.SetCurrentNode(badNode)
		eventEnter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		res := tree.inputCapture(eventEnter)
		assert.Equal(t, eventEnter, res)
	})

	t.Run("inputCapture_KeyRune_GlobalHandled", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		store := newMockStoreWithRootTitle(t, url.URL{Scheme: "mock", Path: "/"}, "Root")
		store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		tree.nav.store = store
		eventRune := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
		res := tree.inputCapture(eventRune)
		assert.Nil(t, res)
	})

	t.Run("inputCapture_KeyRune_SpaceIgnored", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.searchPattern = ""
		eventRune := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
		res := tree.inputCapture(eventRune)
		assert.Equal(t, eventRune, res)
		assert.Equal(t, "", tree.searchPattern)
	})

	t.Run("SetSearch", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.SetSearch("test")
	})

	t.Run("highlightTreeNodes_skipsRoot", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		dirContext := files.NewDirContext(tree.nav.store, "/", nil)
		tree.setCurrentDir(dirContext)
	})

	t.Run("setCurrentDir_Nil", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		tree.setCurrentDir(nil)
	})

	t.Run("setDirContext", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
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
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		root := tree.tv.GetRoot()
		tree.setError(root, context.DeadlineExceeded)
	})

	t.Run("getNodePath", func(t *testing.T) {
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
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
		t.Parallel()
		nav, _, _ := newNavigatorForTest(t)
		tree := NewTree(nav)
		node := tview.NewTreeNode("bad")
		node.SetReference("bad")
		tree.tv.SetCurrentNode(node)
		entry := tree.GetCurrentEntry()
		assert.Nil(t, entry)
	})
}
