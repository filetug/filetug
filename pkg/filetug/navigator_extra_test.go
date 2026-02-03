package filetug

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/viewers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func getDirSummarySafe(nav *Navigator) *viewers.DirPreviewer {
	if nav.previewer == nil {
		return nil
	}
	return nav.previewer.dirPreviewer
}

func TestNavigator_SetStore(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	newStore := osfile.NewStore("/tmp")
	nav.SetStore(newStore)
	assert.Equal[any](t, newStore, nav.store)
}

func TestNavigator_SetFocusToContainer(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectSetFocusMinMaxTimes(app, 0, 1)
	// Test index 1 (files)
	nav.SetFocusToContainer(1)
	// We can't easily check what's focused in tview.Application without running it,
	// but we can ensure it doesn't panic and covers the code.
}

func TestNewNavigator_InvalidURL(t *testing.T) {
	t.Parallel()
	oldGetState := getState
	defer func() { getState = oldGetState }()

	t.Run("Invalid_HTTPS_URL", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{
				CurrentDir: "https:// invalid-url",
			}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
	})
}

func TestNavigator_InputCapture_Extra(t *testing.T) {
	t.Parallel()

	t.Run("AltX", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		app.EXPECT().Stop()
		event := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt)
		res := nav.inputCapture(event)
		assert.Equal(t, (*tcell.EventKey)(nil), res)
	})

	nav, _, _ := newNavigatorForTest(t)

	t.Run("PlainRune", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
		res := nav.inputCapture(event)
		assert.Equal(t, event, res)
	})

	t.Run("AltUnknown", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModAlt)
		res := nav.inputCapture(event)
		assert.Equal(t, event, res)
	})
}

func TestNavigator_Resize_Extra(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)

	t.Run("Resize_ActiveCol0", func(t *testing.T) {
		nav.activeCol = 0
		nav.resize(increase)
		assert.Equal(t, defaultProportions[0]+2, nav.proportions[0])
	})

	t.Run("Resize_ActiveCol1", func(t *testing.T) {
		copy(nav.proportions, defaultProportions)
		nav.activeCol = 1
		nav.resize(increase)
		assert.Equal(t, defaultProportions[1]+2, nav.proportions[1])
	})

	t.Run("Resize_ActiveCol2", func(t *testing.T) {
		copy(nav.proportions, defaultProportions)
		nav.activeCol = 2
		nav.resize(increase)
		assert.Equal(t, defaultProportions[2]+2, nav.proportions[2])
	})
}

func TestNavigator_SetBreadcrumbs_EmptyRelative(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	nav.store = newMockStoreWithRootTitle(t, url.URL{Scheme: "file", Path: "/root"}, "Root")
	nav.current.SetDir(files.NewDirContext(nav.store, "/root", nil))

	nav.setBreadcrumbs()
	// Should return early after pushing root breadcrumb
}

func TestNavigator_SetBreadcrumbs_NoCurrentDir(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	nav.store = newMockStoreWithRoot(t, url.URL{Path: "/"})
	nav.current.SetDir(nil)
	nav.setBreadcrumbs()
}

func TestNavigator_DirSummary_FocusLeft(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)

	expectSetFocusMinMaxTimes(app, 0, 1)

	event := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res := getDirSummarySafe(nav).InputCapture(event)

	assert.Equal(t, (*tcell.EventKey)(nil), res)
}

func TestNavigator_UpdateGitStatus_RealCall(t *testing.T) {
	t.Parallel()
	// This is hard to test without real git, but we can at least try to call it
	// and see it doesn't crash.
	nav, _, _ := newNavigatorForTest(t)
	node := tview.NewTreeNode("test")
	ctx := context.Background()

	// Try a path that definitely exists (project root)
	cwd, _ := os.Getwd()
	nav.updateGitStatus(ctx, nil, cwd, node, "prefix: ")

	// Wait a bit as it might be doing something
	time.Sleep(200 * time.Millisecond)

	// Coverage for case where it's already in cache (first call should have cached it if it's a git repo)
	nav.updateGitStatus(ctx, nil, cwd, node, "prefix: ")
}

func TestNavigator_ShowNodeError_Extra(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	expectQueueUpdateDrawSyncMinMaxTimes(app, 0, 1) // TODO: Make deterministic
	nav.right = NewContainer(2, nav)
	nav.previewer = newPreviewerPanel(nav)

	nodeContext := files.NewDirContext(nil, "/test", nil)
	node := tview.NewTreeNode("test").SetReference(nodeContext)
	nav.showNodeError(node, os.ErrNotExist)
	assert.Equal(t, "file does not exist", nav.previewer.textView.GetText(true))
}

func TestNavigator_ShowDir_GitStatusCall(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	if nav == nil {
		t.Fatal("navigator is nil")
	}
	nav.store = osfile.NewStore("/")

	node := tview.NewTreeNode("test")
	ctx := context.Background()

	// This should trigger go nav.updateGitStatus
	dirContext := files.NewDirContext(nav.store, "/", nil)
	nav.showDir(ctx, node, dirContext, true)
	time.Sleep(100 * time.Millisecond)
}

func TestNewNavigator_EmptyState(t *testing.T) {
	t.Parallel()
	oldGetState := getState
	defer func() { getState = oldGetState }()

	t.Run("Empty_State", func(t *testing.T) {
		getState = func() (*ftstate.State, error) {
			return &ftstate.State{}, nil
		}
		nav, _, _ := newNavigatorForTest(t)
		assert.True(t, nav != nil)
		assert.Equal(t, "file:", nav.store.RootURL().Scheme+":")
	})
}
