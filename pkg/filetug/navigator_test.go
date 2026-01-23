package filetug

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestOnMoveFocusUp(t *testing.T) {
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

		nav.activeCol = -1
		nav.GetInputCapture()(altKey('+'))
		nav.GetInputCapture()(altKey('-'))

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
	app := tview.NewApplication()
	nav := NewNavigator(app)
	if nav == nil {
		t.Fatal("nav is nil")
	}
	node := tview.NewTreeNode("test")

	drawUpdatesCount := 0
	oldQueueUpdateDraw := nav.queueUpdateDraw
	defer func() {
		nav.queueUpdateDraw = oldQueueUpdateDraw
	}()
	nav.queueUpdateDraw = func(f func()) {
		drawUpdatesCount++
	}

	// Use background context for tests
	ctx := context.Background()

	// 1. Not in cache, git status returns nil
	nav.updateGitStatus(ctx, nil, "/non-existent", node, "prefix: ")

	// 2. In cache
	nav.gitStatusCache["/cached"] = &gitutils.RepoStatus{Branch: "main"}
	nav.updateGitStatus(ctx, nil, "/cached", node, "prefix: ")

	// 3. Cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	nav.updateGitStatus(cancelledCtx, nil, "/any", node, "prefix: ")

	time.Sleep(100 * time.Millisecond)
}

func TestNavigator_goDir(t *testing.T) {
	saveCurrentDir = func(string, string) {}
	app := tview.NewApplication()
	nav := NewNavigator(app, OnMoveFocusUp(func(source tview.Primitive) {}))

	t.Run("goDir_Success", func(t *testing.T) {
		nav.goDir(".")
	})

	t.Run("goDir_NonExistent", func(t *testing.T) {
		nav.goDir("/non-existent-Path-12345")
	})

	t.Run("Extra", func(t *testing.T) {
		nav.SetFocusToContainer(0)
		nav.SetFocusToContainer(1)
		nav.SetFocusToContainer(2)
		nav.showMasks()
	})

	t.Run("onDataLoaded_showNodeError", func(t *testing.T) {
		node := tview.NewTreeNode("test").SetReference("/test")
		dirContext := &DirContext{
			Path:     "/test",
			children: []os.DirEntry{mockDirEntry{name: "file.txt", isDir: false}},
		}

		nav.onDataLoaded(node, dirContext, true)
		nav.onDataLoaded(node, dirContext, false)

		err := errors.New("test error")
		nav.showNodeError(node, err)
		nav.showNodeError(nil, err)
	})
}

func TestNavigator_onDataLoaded_isTreeRootChanged(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	node := tview.NewTreeNode("test").SetReference("/test")
	dirContext := &DirContext{
		Path:     "/test",
		children: []os.DirEntry{mockDirEntry{name: "file.txt", isDir: false}},
	}
	nav.onDataLoaded(node, dirContext, true)
}

func TestNavigator_setBreadcrumbs_Complex(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	if nav == nil {
		t.Fatal("nav is nil")
	}

	// Mock store with specific root
	mStore := &mockNavigatorStore{
		rootURL: url.URL{Scheme: "file", Path: "/root/dir"},
	}
	nav.store = mStore
	nav.current.dir = "/root/dir/subdir/subsubdir"

	nav.setBreadcrumbs()

	// Test empty path item
	nav.current.dir = "/root/dir/subdir//subsubdir"
	nav.setBreadcrumbs()

	// Test breadcrumb actions by simulating Enter key on breadcrumbs
	nav.breadcrumbs.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
}

func TestNewNavigator_States(t *testing.T) {
	oldGetState := getState
	defer func() { getState = oldGetState }()

	app := tview.NewApplication()

	t.Run("HTTP_State", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:      "http://example.com",
				CurrentDir: "http://example.com/path",
			}, nil
		}
		nav := NewNavigator(app)
		assert.True(t, nav != nil)
	})

	t.Run("FTP_State", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:      "ftp://example.com",
				CurrentDir: "/path",
			}, nil
		}
		nav := NewNavigator(app)
		assert.True(t, nav != nil)
	})

	t.Run("File_State_With_Entry", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store:           "file:",
				CurrentDir:      "/tmp",
				CurrentDirEntry: "test.txt",
			}, nil
		}
		nav := NewNavigator(app)
		assert.True(t, nav != nil)
		assert.Equal(t, "/tmp", nav.current.dir)
	})

	t.Run("HTTPS_State_Prefix", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				CurrentDir: "https://example.com/path",
			}, nil
		}
		nav := NewNavigator(app)
		assert.True(t, nav != nil)
	})

	t.Run("HTTP_State_No_Schema", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				Store: "http",
			}, nil
		}
		nav := NewNavigator(app)
		assert.True(t, nav != nil)
	})
}

type mockNavigatorStore struct {
	files.Store
	rootURL url.URL
}

func (m *mockNavigatorStore) RootURL() url.URL {
	return m.rootURL
}

func (m *mockNavigatorStore) RootTitle() string {
	return "Root"
}

func (m *mockNavigatorStore) ReadDir(ctx context.Context, path string) ([]os.DirEntry, error) {
	_, _ = ctx, path
	return nil, nil
}

func (m *mockNavigatorStore) CreateDir(ctx context.Context, path string) error {
	_, _ = ctx, path
	return nil
}

func (m *mockNavigatorStore) CreateFile(ctx context.Context, path string) error {
	_, _ = ctx, path
	return nil
}

func TestNavigator_updateGitStatus_Success(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	node := tview.NewTreeNode("test")

	// Mock git status
	ctx := context.Background()
	// Using a path that likely doesn't exist but we want to see it NOT return early if possible.
	// Actually we want to cover the case where status is NOT nil.
	// We can't easily mock gitutils without changing it, but we can try to find a real git repo.
	// Or we can just test the "app == nil" branch which is easy.

	t.Run("NoApp", func(t *testing.T) {
		nav.app = nil
		status := &gitutils.RepoStatus{Branch: "main"}
		// Path matches repo root to ensure status is shown even if clean
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
		nav.app = app
		oldQueueUpdateDraw := nav.queueUpdateDraw
		nav.queueUpdateDraw = func(f func()) {
			f()
		}
		defer func() { nav.queueUpdateDraw = oldQueueUpdateDraw }()

		status := &gitutils.RepoStatus{Branch: "main"}
		// Path matches repo root to ensure status is shown even if clean
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
}

func TestNavigator_showDir_FileScheme(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	nav.queueUpdateDraw = func(f func()) {
		f()
	}
	node := tview.NewTreeNode("test")

	// Mock store with file scheme
	mStore := &mockNavigatorStore{
		rootURL: url.URL{Scheme: "file", Path: "/"},
	}
	nav.store = mStore

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nav.showDir(ctx, node, "/tmp", true)
	time.Sleep(100 * time.Millisecond) // Give some time for goroutine
}
