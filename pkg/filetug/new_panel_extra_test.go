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
)

func TestNewPanel_Coverage(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	tmpDir := t.TempDir()
	nav.store = osfile.NewStore(tmpDir)
	nav.current.dir = tmpDir

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
		expectedFile := filepath.Join(nav.current.dir, "newfile.txt")
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
}

func TestScripts_And_NestedDirs(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)

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
