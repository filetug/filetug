package tui

import (
	"path"

	tea "charm.land/bubbletea/v2"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

var _ nav.KeyCapturer = browser{}

// CapturesKey claims every key while a dialog is open, except Ctrl+C, which the
// shell keeps for quitting.
func (b browser) CapturesKey(msg tea.KeyPressMsg) bool {
	return b.modal != nil && msg.String() != "ctrl+c"
}

// openDialog shows a dialog above the browser and takes the focus for it.
func (b browser) openDialog(d *dialog) (nav.Screen, tea.Cmd) {
	d.box.SetMaxSize(b.w, b.h)
	b.modal = d
	return b, nav.SetFocus(nav.FocusToContent)
}

// dialogDoneMsg is a dialog's answer. The shell keeps widgets.ModalDoneMsg for
// its own alerts, so the answer of a dialog is renamed on its way out.
type dialogDoneMsg widgets.ModalDoneMsg

// modalKey gives a key to the dialog.
func (b browser) modalKey(msg tea.KeyPressMsg) (nav.Screen, tea.Cmd) {
	var cmd tea.Cmd
	b.modal.box, cmd = b.modal.box.Update(msg)
	return b, renameDone(cmd)
}

// renameDone turns the ModalDoneMsg a command delivers into a dialogDoneMsg.
func renameDone(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if done, ok := msg.(widgets.ModalDoneMsg); ok {
			return dialogDoneMsg(done)
		}
		return msg
	}
}

// modalDone closes the dialog. A help dialog returns the focus to the tree;
// the first button of a delete dialog deletes.
func (b browser) modalDone(msg dialogDoneMsg) (nav.Screen, tea.Cmd) {
	d := b.modal
	if d == nil || d.id != msg.ID {
		return b, nil
	}
	b.modal = nil
	switch {
	case d.kind == dialogHelp:
		return b, nav.SetFocus(nav.FocusToMenu)
	case msg.Index == 0:
		return b, deleteCmd(b.sess.store, d.path)
	}
	return b, nil
}

// deleteFocused is F8: ask to delete the entry under the file list's cursor.
// The tree answers the same message when it has the focus.
func (b browser) deleteFocused() (nav.Screen, tea.Cmd) {
	if !b.focused || b.focus != paneFiles {
		return b, nil
	}
	ref, ok := b.files.CurrentRef()
	if !ok || ref.Parent {
		return b, nil
	}
	return b.openDialog(newDeleteDialog(ref.Entry.Name(), ref.Entry.FullName()))
}

// deleted reports a failed deletion, or shows the directory again without the
// deleted entry.
func (b browser) deleted(msg deletedMsg) (nav.Screen, tea.Cmd) {
	if msg.Err != nil {
		return b, nav.Alert("Delete failed", msg.Err.Error(), 0, nav.FocusToKeep)
	}
	if msg.Path == b.sess.currentPath() {
		return b, goDir(b.sess.treeRoot.Path()) // the directory shown is gone
	}
	return b, b.reload()
}

// reload reads the directory being shown again. The tree is reloaded when the
// file list shows its root, so that a deleted or created directory appears.
func (b browser) reload() tea.Cmd {
	s := b.sess
	if s.current == nil || s.treeRoot == nil || s.current.Path() == s.treeRoot.Path() {
		return goDir(s.currentPath())
	}
	return widgets.Emit(showDirMsg{Path: s.currentPath(), Force: true})
}

// openPanel puts a panel in place of the preview and focuses it.
func (b browser) openPanel(msg openPanelMsg) (nav.Screen, tea.Cmd) {
	b.panel = newPanelFor(msg.Kind)
	b.focus = panePreview
	b.layout()
	b.applyFocus()
	return b, nav.SetFocus(nav.FocusToContent)
}

// panelClosed restores the preview and moves the focus.
func (b browser) panelClosed(msg panelClosedMsg) (nav.Screen, tea.Cmd) {
	b.panel = nil
	b.focus = paneFiles
	b.layout()
	b.applyFocus()
	return b, nav.SetFocus(msg.To)
}

// createEntry creates the directory or file a panel asked for in the directory
// being shown.
func (b browser) createEntry(msg createEntryMsg) (nav.Screen, tea.Cmd) {
	dir := b.sess.currentPath()
	if msg.Name == "" || dir == "" {
		return b, nil
	}
	return b, createCmd(b.sess.store, msg.Dir, path.Join(dir, msg.Name), msg.Name)
}

// created opens a new directory, or shows the directory with the new file
// selected.
func (b browser) created(msg createdMsg) (nav.Screen, tea.Cmd) {
	if msg.Err != nil {
		return b, nav.Alert("Could not create "+msg.Name, msg.Err.Error(), 0, nav.FocusToKeep)
	}
	b.panel = nil
	b.focus = paneFiles
	b.layout()
	b.applyFocus()
	if msg.Dir {
		return b, tea.Batch(nav.SetFocus(nav.FocusToContent), goDir(msg.Path))
	}
	b.sess.entryName = msg.Name
	return b, tea.Batch(nav.SetFocus(nav.FocusToContent), widgets.Emit(showDirMsg{Path: b.sess.currentPath(), Force: true}))
}

// toPanel delivers the answer of a panel's form or list to the panel.
func (b browser) toPanel(msg tea.Msg) (nav.Screen, tea.Cmd) {
	if b.panel == nil {
		return b, nil
	}
	cmd := b.panel.Update(msg)
	return b, cmd
}
