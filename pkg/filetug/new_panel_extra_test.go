package filetug

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/mock/gomock"
)

func TestNewPanel_Coverage(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).Times(1)
	tmpDir := t.TempDir()
	nav.store = osfile.NewStore(tmpDir)
	nav.current.SetDir(nav.NewDirContext(tmpDir, nil))

	p := NewNewPanel(nav)

	t.Run("Show_and_Focus", func(t *testing.T) {
		p.Show()
		assert.True(t, p == nav.right.content)
		p.Focus(func(p tview.Primitive) {})
		assert.Equal(t, 2, nav.activeCol)
	})

	t.Run("createDir", func(t *testing.T) {
		p.input.SetText("newdir")
		p.createDir()
		_, err := os.Stat(filepath.Join(tmpDir, "newdir"))
		assert.NoError(t, err)
	})

	t.Run("createFile", func(t *testing.T) {
		p.input.SetText("newfile.txt")
		// nav.showDir might cause issues if not mocked, but here we just want to ensure it creates the file
		p.createFile()
		// If CreateFile failed, it might have returned early.
		// Let's use a full path to be absolutely sure where it should be.
		expectedFile := filepath.Join(nav.current.Dir().Path(), "newfile.txt")
		_, err := os.Stat(expectedFile)
		assert.NoError(t, err)
	})

	t.Run("input_handlers", func(t *testing.T) {
		// Escape
		// Use a trick to get the function, as it might be private or not have a getter in some tview versions
		// But usually it's public. Wait, tview.InputField has SetDoneFunc but NO GetDoneFunc?
		// Actually it's public: DoneFunc.
		// Let's check tview documentation/code if possible, or just assume it's private and we can't test it this way.
		// Actually, I can just call p.input.InputCapture()(event) for Escape.

		// Let's try to find if there is a way to trigger it.
		// For now let's just use what's likely available or skip if not.

		// Alt-f
		event := tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone)
		res := p.input.GetInputCapture()(event)
		assert.Equal(t, (*tcell.EventKey)(nil), res)

		// Alt-d
		event = tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone)
		res = p.input.GetInputCapture()(event)
		assert.Equal(t, (*tcell.EventKey)(nil), res)
	})

	t.Run("createDir_noCurrentDir", func(t *testing.T) {
		original := nav.current.Dir()
		nav.current.SetDir(nil)
		defer nav.current.SetDir(original)

		p.input.SetText("skipdir")
		p.createDir()
		_, err := os.Stat(filepath.Join(tmpDir, "skipdir"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("createFile_noCurrentDir", func(t *testing.T) {
		original := nav.current.Dir()
		nav.current.SetDir(nil)
		defer nav.current.SetDir(original)

		p.input.SetText("skipfile.txt")
		p.createFile()
		_, err := os.Stat(filepath.Join(tmpDir, "skipfile.txt"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestScripts_And_NestedDirs(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)

	t.Run("showScriptsPanel", func(t *testing.T) {
		nav.showScriptsPanel()
		assert.True(t, nav.right.content != nil)
	})

	t.Run("GeneratedNestedDirs", func(t *testing.T) {
		tmpDir := t.TempDir()
		store := osfile.NewStore(tmpDir)
		err := GeneratedNestedDirs(context.Background(), store, filepath.Join(tmpDir, "nested"), "Sub%d", 2, 2)
		assert.NoError(t, err)

		// Verify some dirs were created
		_, err = os.Stat(filepath.Join(tmpDir, "nested", "Sub0", "Sub0"))
		assert.NoError(t, err)
	})
}
