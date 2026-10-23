package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// dialogKind says what a dialog is for.
type dialogKind int

const (
	dialogHelp dialogKind = iota
	dialogDelete
)

// helpText lists the keys of the application.
const helpText = `F1         Help
F7         New directory or file
F8         Delete the selected directory or file
F10        Scripts
Alt+F      Favorites
Alt+M      Masks
Alt+W      Git worktrees with optional WB metadata
Alt+/      Go to the root directory
Alt+~      Go to the home directory
Alt+=      Widen the focused panel
Alt+-      Narrow the focused panel
Alt+0      Reset the panel sizes
Alt+X      Exit the app`

// dialog is a modal box above the browser.
type dialog struct {
	id   string
	kind dialogKind
	box  widgets.Modal
	// path is what a delete dialog deletes.
	path string
}

func newHelpDialog() *dialog {
	d := &dialog{id: "help", kind: dialogHelp, box: widgets.NewModal("help")}
	d.box.SetTitle("FileTug - Help")
	d.box.SetText(helpText)
	d.box.SetButtons("Close")
	return d
}

func newDeleteDialog(name, fullPath string) *dialog {
	d := &dialog{id: "delete", kind: dialogDelete, box: widgets.NewModal("delete"), path: fullPath}
	d.box.SetTitle("Delete")
	d.box.SetText("Delete " + name + "?")
	d.box.SetButtons("Delete", "Cancel")
	d.box.SetColors(widgets.ModalColors{Border: theme.ErrorColor()})
	return d
}

// confirmDeleteMsg asks to confirm the deletion of a directory or file.
type confirmDeleteMsg struct{ Name, Path string }

// deleteMsg is F8: delete what is selected in the focused list.
type deleteMsg struct{}

// deletedMsg is the result of a deletion.
type deletedMsg struct {
	Path string
	Err  error
}

// deleteCmd deletes a path off the event loop.
func deleteCmd(store files.Store, fullPath string) tea.Cmd {
	return func() tea.Msg {
		return deletedMsg{Path: fullPath, Err: store.Delete(context.Background(), fullPath)}
	}
}

// createdMsg is the result of creating a directory or a file.
type createdMsg struct {
	Dir  bool
	Path string
	Name string
	Err  error
}

// createCmd creates a directory or an empty file off the event loop.
func createCmd(store files.Store, dir bool, fullPath, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		if dir {
			err = store.CreateDir(ctx, fullPath)
		} else {
			err = store.CreateFile(ctx, fullPath)
		}
		return createdMsg{Dir: dir, Path: fullPath, Name: name, Err: err}
	}
}
