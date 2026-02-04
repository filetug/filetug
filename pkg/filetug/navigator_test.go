package filetug

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/tviewmocks"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/mock/gomock"
)

func TestOnMoveFocusUp(t *testing.T) {
	t.Parallel()
	var s tview.Primitive
	f := func(source tview.Primitive) {
		s = source
	}
	o := OnMoveFocusUp(f)
	var options navigatorOptions
	o(&options)
	assert.Equal(t, f, options.moveFocusUp)

	textView := tview.NewTextView()
	options.moveFocusUp(textView)
	assert.Equal[tview.Primitive](t, textView, s)
}

func TestNavigator(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	app := tviewmocks.NewMockApp(ctrl)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))
	if nav == nil {
		t.Fatal("nav is nil")
	}

	t.Run("SetFocus", func(t *testing.T) {
		app.EXPECT().SetFocus(gomock.Any()).Times(1)
		nav.SetFocus()
	})

	altKey := func(r rune) *tcell.EventKey {
		return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModAlt)
	}

	for activeCol := 0; activeCol < 3; activeCol++ {
		for _, r := range []rune{
			'+',
			'0',
			'-',
		} {
			t.Run(fmt.Sprintf("col=%d;r=%c", activeCol, r), func(t *testing.T) {
				nav.GetInputCapture()(altKey(r))
			})
		}
	}

	for _, r := range []rune{
		'f',
		'm',
		'r',
		'h',
		'?',
		'z',
	} {
		t.Run(string(r), func(t *testing.T) {
			app.EXPECT().SetFocus(gomock.Any()).AnyTimes()
			app.EXPECT().SetRoot(gomock.Any(), true).AnyTimes()
			nav.GetInputCapture()(altKey(r))
		})
	}

	t.Run("NavigatorInputCapture", func(t *testing.T) {
		app.EXPECT().SetFocus(gomock.Any()).AnyTimes()
		app.EXPECT().SetRoot(gomock.Any(), true).AnyTimes()
		nav.GetInputCapture()(altKey('f'))
		nav.GetInputCapture()(altKey('m'))
		nav.GetInputCapture()(altKey('r'))
		nav.GetInputCapture()(altKey('h'))
		nav.GetInputCapture()(altKey('?'))
		nav.GetInputCapture()(altKey('z')) // unknown alt key

		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

		// Test moveFocusUp in navigator
		nav.o.moveFocusUp(nav.files)

		// Test Ctrl modifier
		nav.GetInputCapture()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl))
		assert.True(t, nav.bottom.isCtrl)
	})
}

func TestNavigator_GitStatus(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	// Expect QueueUpdateDraw ONLY if we expect it to be called.
	// 1. /non-existent: getGitStatus returns nil because GetRepositoryRoot fails. No call.
	// 2. /cached: getGitStatus returns from cache. But gitStatusText returns empty because GetRepositoryRoot fails.
	// 3. /any: context cancelled. No call.

	if nav == nil {
		t.Fatal("nav is nil")
	}
	node := tview.NewTreeNode("test")

	// Use background context for tests
	ctx := context.Background()

	// 1. Not in cache, git status returns nil
	nav.updateGitStatus(ctx, nil, "/non-existent", node, "prefix: ")

	nav.gitStatusCacheMu.Lock()
	nav.gitStatusCache["/cached"] = &gitutils.RepoStatus{Branch: "main"}
	nav.gitStatusCacheMu.Unlock()
	nav.updateGitStatus(ctx, nil, "/cached", node, "prefix: ")

	// 3. Cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	nav.updateGitStatus(cancelledCtx, nil, "/any", node, "prefix: ")

	time.Sleep(100 * time.Millisecond)
}

func TestNavigator_goDir(t *testing.T) {
	//withTestGlobalLock(t)

	//ctrl := gomock.NewController(t)
	//app := navigator.NewMockApp(ctrl)

	//nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))

	t.Run("goDir_Success", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		expectQueueUpdateDrawSyncMinMaxTimes(app, 0, 5)
		dirContext := nav.NewDirContext(".", nil)
		nav.goDir(dirContext)
		assert.True(t, saveCurrentDirCalled)
	})

	t.Run("goDir_NonExistent", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		//expectSetFocusTimes(app, 1)
		expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 2)
		dirContext := nav.NewDirContext("/non-existent-Dir-12345", nil)
		nav.goDir(dirContext)
		assert.True(t, saveCurrentDirCalled)
	})

	t.Run("goDir_Nil", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		app.EXPECT().QueueUpdateDraw(gomock.Any()).MaxTimes(1)
		nav.goDir(nil)
		assert.False(t, saveCurrentDirCalled)
	})

	t.Run("Extra", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		expectSetFocusMinMaxTimes(app, 1, 3)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		nav.SetFocusToContainer(0)
		nav.SetFocusToContainer(1)
		nav.SetFocusToContainer(2)
		nav.showMasks()
		assert.False(t, saveCurrentDirCalled)
	})

	t.Run("onDataLoaded_showNodeError", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 5)
		nodeContext := nav.NewDirContext("/test", nil)
		node := tview.NewTreeNode("test").SetReference(nodeContext)
		dirContext := nav.NewDirContext("/test", []os.DirEntry{mockDirEntry{
			name: "file.txt", isDir: false}})

		ctx := context.Background()
		nav.onDataLoaded(ctx, node, dirContext, true)
		//nav.onDataLoaded(ctx, node, dirContext, false)

		err := errors.New("test error")
		nav.showNodeError(node, err)
		//nav.showNodeError(nil, err)
		assert.False(t, saveCurrentDirCalled)
	})

	t.Run("onDataLoaded_updatesPreviewer", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		saveCurrentDirCalled := false
		nav.saveCurrentDir = func(string, string) {
			saveCurrentDirCalled = true
		}
		queueUpdateDrawCount := 0
		app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes().DoAndReturn(func(f func()) {
			f()
			queueUpdateDrawCount++
		})
		//expectQueueUpdateDrawSyncMinMaxTimes(app, 0, 20)
		tempDir := t.TempDir()
		childDir := filepath.Join(tempDir, "child")
		if err := os.Mkdir(childDir, 0755); err != nil {
			t.Fatalf("failed to create child dir: %v", err)
		}
		nav.previewer.SetTitle("initial")
		nodeContext := nav.NewDirContext(childDir, nil)
		node := tview.NewTreeNode("temp").SetReference(nodeContext)
		dirContext := files.NewDirContext(osfile.NewStore("/"), childDir,
			[]os.DirEntry{mockDirEntry{name: "file.txt", isDir: false}})
		ctx := context.Background()
		nav.onDataLoaded(ctx, node, dirContext, false)
		assert.NotEqual(t, "initial", nav.previewer.GetTitle())
		assert.False(t, saveCurrentDirCalled)
		t.Logf("queueUpdateDrawCount=%d", queueUpdateDrawCount)
	})
	time.Sleep(time.Second)
}

func TestNavigator_goDir_TreeRootChangeRefreshesChildren(t *testing.T) {
	t.Parallel()
	withTestGlobalLock(t)
	oldGetState := getState
	getState = func() (*ftstate.State, error) { return nil, nil }
	defer func() {
		getState = oldGetState
	}()

	oldSaveCurrentDir := saveCurrentDir
	saveCurrentDir = func(string, string) {}
	defer func() {
		saveCurrentDir = oldSaveCurrentDir
	}()

	nav, app, ctrl := newNavigatorForTest(t)
	entries := []os.DirEntry{
		mockDirEntry{name: "child", isDir: true},
	}
	store := files.NewMockStore(ctrl)
	store.EXPECT().RootURL().Return(url.URL{Scheme: "mock", Path: "/root"}).AnyTimes()
	store.EXPECT().RootTitle().Return("Mock").AnyTimes()
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p string) ([]os.DirEntry, error) {
			if p == "/root" {
				return entries, nil
			}
			return []os.DirEntry{}, nil
		},
	).AnyTimes()
	nav.SetStore(store)
	dirContext := nav.NewDirContext("/root", entries)
	nav.current.SetDir(dirContext)
	nav.dirsTree.rootNode.SetReference(dirContext)

	app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes().DoAndReturn(func(f func()) {
		f()
	})

	nav.onDataLoaded(context.Background(), nav.dirsTree.rootNode, dirContext, true)

	// In TestNavigator_goDir_TreeRootChangeRefreshesChildren we want to test that
	// root node refresh adds children. By calling onDataLoaded directly with isTreeRootChanged=true,
	// we bypass the goroutine in showDir but test the core logic of refreshing the tree.
	children := nav.dirsTree.rootNode.GetChildren()
	if len(children) == 0 {
		t.Fatal("expected tree children after goDir refresh logic")
	}
}

func TestNavigator_showDir_UsesRequestedPathForAsyncLoad(t *testing.T) {
	t.Parallel()
	withTestGlobalLock(t)
	oldGetState := getState
	getState = func() (*ftstate.State, error) { return nil, nil }
	defer func() {
		getState = oldGetState
	}()

	nav, _, _ := newNavigatorForTest(t)

	firstEntries := []os.DirEntry{mockDirEntry{name: "firstChild", isDir: true}}
	secondEntries := []os.DirEntry{mockDirEntry{name: "secondChild", isDir: true}}
	seen := make(chan string, 2)
	store := newMockStore(t)
	store.EXPECT().RootURL().Return(url.URL{Scheme: "mock", Path: "/"}).AnyTimes()
	store.EXPECT().RootTitle().Return("Mock").AnyTimes()
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, path string) ([]os.DirEntry, error) {
			seen <- path
			if path == "/first" {
				return firstEntries, nil
			}
			if path == "/second" {
				return secondEntries, nil
			}
			return nil, nil
		},
	).AnyTimes()
	nav.store = store

	ctx := context.Background()
	nodeFirst := tview.NewTreeNode("first")
	nodeSecond := tview.NewTreeNode("second")

	nav.showDir(ctx, nodeFirst, nav.NewDirContext("/first", nil), true)
	nav.showDir(ctx, nodeSecond, nav.NewDirContext("/second", nil), true)
	deadline := time.Now().Add(2 * time.Second)
	var lastSeen string
	for time.Now().Before(deadline) {
		select {
		case lastSeen = <-seen:
			if lastSeen == "/first" || lastSeen == "/second" {
				return
			}
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatalf("timeout waiting for /first or /second; last seen %q", lastSeen)
}

func TestNavigator_onDataLoaded_isTreeRootChanged(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	nodeContext := nav.NewDirContext("/test", nil)
	node := tview.NewTreeNode("test").SetReference(nodeContext)
	dirContext := files.NewDirContext(nil, "/test", []os.DirEntry{mockDirEntry{name: "file.txt", isDir: false}})
	ctx := context.Background()
	nav.onDataLoaded(ctx, node, dirContext, true)
}

func TestNavigator_setBreadcrumbs_Complex(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 2) // TODO: Make deterministic
	if nav == nil {
		t.Fatal("nav is nil")
	}

	// Mock store with specific root
	store := newMockStoreWithRootTitle(t, url.URL{Scheme: "file", Path: "/root/dir"}, "Root")
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	nav.store = store
	nav.current.SetDir(nav.NewDirContext("/root/dir/subdir/subsubdir", nil))

	nav.setBreadcrumbs()

	// Test empty path item
	nav.current.SetDir(nav.NewDirContext("/root/dir/subdir//subsubdir", nil))
	nav.setBreadcrumbs()

	// Test breadcrumb actions by simulating Enter key on breadcrumbs
	nav.breadcrumbs.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
}

func TestNewNavigator_States(t *testing.T) {
	withTestGlobalLock(t)
	oldGetState := getState
	defer func() { getState = oldGetState }()

	t.Run("HTTP_State", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:      "http://example.com",
				CurrentDir: "http://example.com/path",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
	})

	t.Run("FTP_State", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:      "ftp://example.com",
				CurrentDir: "/path",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
	})

	t.Run("File_State_With_Entry", func(t *testing.T) {
		tempDir := t.TempDir()
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:           "file:",
				CurrentDir:      tempDir,
				CurrentDirEntry: "test.txt",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		nav.store = osfile.NewStore(tempDir)
		initNavigatorWithPersistedState(nav)
		assert.True(t, nav != nil)
		if nav.current.Dir() == nil {
			t.Fatal("Current dir is nil")
		}
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if nav.current.Dir() != nil && nav.current.Dir().Path() == tempDir {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		currentPath := nav.current.Dir().Path()
		assert.True(t, currentPath == tempDir || currentPath == "/")
	})

	t.Run("HTTPS_State_Prefix", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				CurrentDir: "https://example.com/path",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
	})

	t.Run("HTTP_State_No_Schema", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store: "http",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
	})
}

func TestNavigator_updateGitStatus_Success(t *testing.T) {
	//withTestGlobalLock(t)

	// Mock git status
	ctx := context.Background()
	// Using a path that likely doesn't exist but we want to see it NOT return early if possible.
	// Actually we want to cover the case where status is NOT nil.
	// We can't easily mock gitutils without changing it, but we can try to find a real git repo.
	// Or we can just test the "app == nil" branch which is easy.

	t.Run("NoApp", func(t *testing.T) {
		nav, _, _ := newNavigatorForTest(t)
		node := tview.NewTreeNode("test")

		status := &gitutils.RepoStatus{Branch: "main"}
		// Dir matches repo root to ensure status is shown even if clean
		path := "/repo"
		oldOsStat := gitutils.OsStat
		gitutils.OsStat = func(name string) (os.FileInfo, error) {
			if name == "/repo/.git" {
				return mockFileInfo{isDir: true}, nil // exists and is a dir
			}
			return oldOsStat(name)
		}
		defer func() { gitutils.OsStat = oldOsStat }()

		nav.gitStatusCache[path] = status
		nav.updateGitStatus(ctx, nil, path, node, "prefix: ")
		assert.Equal(t, "prefix: "+status.String(), node.GetText())
	})

	t.Run("WithAppCached", func(t *testing.T) {
		nav, _, _ := newNavigatorForTest(t)
		node := tview.NewTreeNode("test")

		status := &gitutils.RepoStatus{Branch: "main"}
		// Dir matches repo root to ensure status is shown even if clean
		path := "/repo"
		oldOsStat := gitutils.OsStat
		gitutils.OsStat = func(name string) (os.FileInfo, error) {
			if name == "/repo/.git" {
				return mockFileInfo{isDir: true}, nil // exists and is a dir
			}
			return oldOsStat(name)
		}
		defer func() { gitutils.OsStat = oldOsStat }()

		nav.gitStatusCache[path] = status
		nav.updateGitStatus(ctx, nil, path, node, "prefix: ")
		assert.Equal(t, "prefix: "+status.String(), node.GetText())
	})

	t.Run("PrefixAlreadyHasStatus", func(t *testing.T) {
		nav, _, _ := newNavigatorForTest(t)
		node := tview.NewTreeNode("test")

		status := &gitutils.RepoStatus{Branch: "main"}
		path := "/repo"
		oldOsStat := gitutils.OsStat
		gitutils.OsStat = func(name string) (os.FileInfo, error) {
			if name == "/repo/.git" {
				return mockFileInfo{isDir: true}, nil
			}
			return oldOsStat(name)
		}
		defer func() { gitutils.OsStat = oldOsStat }()

		nav.gitStatusCache[path] = status
		prefixWithStatus := "prefix: " + status.String()
		nav.updateGitStatus(ctx, nil, path, node, prefixWithStatus)
		expected := "prefix: " + status.String()
		actual := node.GetText()
		assert.Equal(t, expected, actual)
	})
}

func TestNavigator_showDir_FileScheme(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).MinTimes(0).MaxTimes(1)
	node := tview.NewTreeNode("test")

	// Mock store with file scheme
	store := newMockStoreWithRootTitle(t, url.URL{Scheme: "file", Path: "/"}, "Root")
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	nav.store = store

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dirContext := nav.NewDirContext("/tmp", nil)
	nav.showDir(ctx, node, dirContext, true)
	time.Sleep(100 * time.Millisecond) // Give some time for goroutine
}

func TestNavigator_showDir_EarlyReturnAndExpandHome(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 3)
	store := newMockStoreWithRootTitle(t, url.URL{Scheme: "file", Path: "/"}, "Root")
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	nav.store = store

	ctx := context.Background()
	nav.showDir(ctx, nil, nil, false)

	nav.current.SetDir(nav.NewDirContext("/tmp", nil))
	sameContext := nav.NewDirContext("/tmp", nil)
	nav.showDir(ctx, nil, sameContext, false)

	homeContext := nav.NewDirContext("~", nil)
	nav.showDir(ctx, nil, homeContext, false)
	time.Sleep(50 * time.Millisecond)
}

func TestNavigator_globalNavInputCapture(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectQueueUpdateDrawSyncMinMaxTimes(app, 1, 6) // TODO: Why 6? Explain with a comment or fix
	store := newMockStoreWithRootTitle(t, url.URL{Scheme: "mock", Path: "/"}, "Root")
	store.EXPECT().ReadDir(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	nav.store = store

	eventSlash := tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)
	res := nav.globalNavInputCapture(eventSlash)
	assert.Equal(t, (*tcell.EventKey)(nil), res)

	eventBacktick := tcell.NewEventKey(tcell.KeyRune, '`', tcell.ModNone)
	res = nav.globalNavInputCapture(eventBacktick)
	assert.Equal(t, (*tcell.EventKey)(nil), res)

	eventOther := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	res = nav.globalNavInputCapture(eventOther)
	assert.Equal(t, eventOther, res)

	eventNonRune := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	res = nav.globalNavInputCapture(eventNonRune)
	assert.Equal(t, eventNonRune, res)
}
