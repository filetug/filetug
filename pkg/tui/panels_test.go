package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func TestHelpDialog(t *testing.T) {
	h := open(t, tree(t))
	h.Press("f1")
	h.RequireContains("FileTug - Help").RequireContains("Alt+W").RequireContains("Close")
	if browserOf(h).modal == nil || h.Model().Zone() != nav.FocusToContent {
		t.Fatal("the dialog takes the focus")
	}
	h.Press("down", "a") // swallowed by the dialog
	h.RequireContains("FileTug - Help")
	h.Press("enter")
	h.RequireNotContains("FileTug - Help")
	if browserOf(h).modal != nil || h.Model().Zone() != nav.FocusToMenu {
		t.Fatal("closing the help returns the focus to the tree")
	}
}

func TestHelpDialogKeepsCtrlCForTheShell(t *testing.T) {
	b := newBrowser(&session{})
	b.modal = newHelpDialog()
	if !b.CapturesKey(tea.KeyPressMsg{Code: 'a', Text: "a"}) || b.CapturesKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}) {
		t.Fatal("keys go to the dialog except Ctrl+C")
	}
	if (browser{}).CapturesKey(tea.KeyPressMsg{Code: 'a', Text: "a"}) {
		t.Fatal("no dialog, no capture")
	}
	s, _ := b.Update(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft})
	if s.(browser).modal == nil {
		t.Fatal("the mouse does not close the dialog")
	}
}

func TestDeleteAFileAfterConfirming(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("right", "down", "down") // ".." then README.md, a.go
	h.Press("f8")
	h.RequireContains("Delete a.go?")
	h.Press("enter")
	if _, err := os.Stat(filepath.Join(dir, "a.go")); !os.IsNotExist(err) {
		t.Fatal("the file is deleted")
	}
	h.RequireNotContains("a.go ").RequireContains("README.md")
}

func TestDeleteCanBeCancelled(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("right", "down", "f8")
	h.Press("right", "enter") // the Cancel button
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatal("cancel keeps the file")
	}
	h.RequireNotContains("Delete README.md?")
	h.Press("f8", "esc")
	if browserOf(h).modal != nil {
		t.Fatal("Esc cancels as well")
	}
}

func TestDeleteADirectoryFromTheTree(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("down", "f8") // Gamma is an empty directory
	h.RequireContains("Delete Gamma?")
	h.Press("enter")
	if _, err := os.Stat(filepath.Join(dir, "Gamma")); !os.IsNotExist(err) {
		t.Fatal("the directory is deleted")
	}
	h.RequireNotContains("Gamma")
}

func TestDeleteNeedsSomethingToDelete(t *testing.T) {
	h := open(t, tree(t))
	h.Press("f8") // the tree's root row
	if browserOf(h).modal != nil {
		t.Fatal("the root row is not deleted from here")
	}
	h.Press("right", "f8") // the ".." row
	if browserOf(h).modal != nil {
		t.Fatal("the parent row is not deleted")
	}
	h.Press("right", "f8") // the preview has the focus
	if browserOf(h).modal != nil {
		t.Fatal("nothing to delete in the preview")
	}
	tr := newTreeScreen(&session{store: osStore()})
	for _, id := range []string{loadingID, errorID} {
		tr.tree.SetRoots(widgets.TreeNode{ID: id, Text: "x"})
		if tr.deleteCurrent() != nil {
			t.Fatalf("%s is not a directory", id)
		}
	}
}

func TestDeleteFailureIsReported(t *testing.T) {
	b := newBrowser(&session{})
	_, cmd := b.Update(deletedMsg{Path: "/x", Err: errors.New("denied")})
	if cmd == nil {
		t.Fatal("the failure is shown in an alert")
	}
	h := open(t, tree(t))
	h.Send(deletedMsg{Path: "/x", Err: errors.New("denied")})
	h.RequireContains("Delete failed").RequireContains("denied")
}

func TestModalDoneForAnotherDialogIsIgnored(t *testing.T) {
	b := newBrowser(&session{})
	if _, cmd := b.Update(dialogDoneMsg{ID: "help"}); cmd != nil {
		t.Fatal("no dialog is open")
	}
	b.modal = newHelpDialog()
	s, _ := b.Update(dialogDoneMsg{ID: "delete"})
	if s.(browser).modal == nil {
		t.Fatal("another dialog's answer does not close this one")
	}
}

func TestNewPanelCreatesAFile(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("f7")
	h.RequireContains("Create directory").RequireContains("Create file")
	if browserOf(h).panel == nil || h.Model().Zone() != nav.FocusToContent {
		t.Fatal("the panel takes the focus")
	}
	h.Type("notes.txt").Press("enter")
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal("Enter in the name creates a file")
	}
	h.RequireContains("notes.txt")
	if browserOf(h).panel != nil || browserOf(h).focus != paneFiles {
		t.Fatal("the panel closes and the file list has the focus")
	}
	if p := browserOf(h).preview.Title(); p != "notes.txt" {
		t.Fatalf("the new file is selected: %q", p)
	}
}

func TestNewPanelCreatesADirectoryWithTheButton(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("f7").Type("fresh").Press("tab", "enter")
	if info, err := os.Stat(filepath.Join(dir, "fresh")); err != nil || !info.IsDir() {
		t.Fatal("the button creates a directory")
	}
	if got := browserOf(h).sess.treeRoot.Path(); got != filepath.Join(dir, "fresh") {
		t.Fatalf("and goes into it: %q", got)
	}
}

func TestNewPanelEscapeAndEmptyName(t *testing.T) {
	h := open(t, tree(t))
	h.Press("f7", "enter") // empty name: nothing happens
	if browserOf(h).panel == nil {
		t.Fatal("an empty name keeps the panel")
	}
	h.Press("esc")
	if browserOf(h).panel != nil || h.Model().Zone() != nav.FocusToMenu {
		t.Fatal("Esc closes the panel and returns to the tree")
	}
	b := newBrowser(&session{})
	if _, cmd := b.Update(createEntryMsg{Name: "x"}); cmd != nil {
		t.Fatal("without a directory nothing is created")
	}
}

func TestCreateFailureIsReported(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("f7").Type("a.go").Press("tab", "enter") // a directory called like a file
	h.RequireContains("Could not create a.go")
}

func TestScriptsPanel(t *testing.T) {
	h := open(t, tree(t))
	h.Press("f10")
	h.RequireContains("Nested Dirs Generator")
	h.Press("enter")
	h.RequireContains("FilesPerDir").RequireContains("File Size (bytes)").RequireContains("Generate")
	h.Press("tab", "tab", "tab", "tab", "enter") // the Generate button does nothing yet
	h.RequireContains("Generate")
	h.Press("tab", "enter") // Cancel
	if browserOf(h).panel != nil || h.Model().Zone() != nav.FocusToContent {
		t.Fatal("Cancel closes the panel")
	}
	h.Press("f10", "esc")
	if browserOf(h).panel != nil {
		t.Fatal("Esc closes the scripts")
	}
	h.Press("f10", "enter", "esc")
	if browserOf(h).panel != nil {
		t.Fatal("Esc closes the generator")
	}
}

func TestMasksPanel(t *testing.T) {
	h := open(t, tree(t))
	h.Press("alt+m")
	h.RequireContains("Coding").RequireContains("Data").RequireContains("CurrDir")
	h.Press("down", "up")
	h.Press("esc")
	if browserOf(h).panel != nil {
		t.Fatal("Esc closes the masks")
	}
}

func TestPanelsCoverTheirPlainAPI(t *testing.T) {
	for _, kind := range []panelKind{panelNew, panelScripts, panelGenerator, panelMasks} {
		p := newPanelFor(kind)
		p.SetSize(40, 10)
		p.SetFocused(true)
		p.SetFocused(false)
		if p.Title() == "" || p.View() == "" {
			t.Errorf("panel %d has a title and a view", kind)
		}
	}
}

func TestPanelGetsClicksAndKeepsTheLayout(t *testing.T) {
	h := open(t, tree(t))
	h.Press("f7")
	h.Click(100, 4) // inside the panel
	if browserOf(h).focus != panePreview || browserOf(h).panel == nil {
		t.Fatal("a click keeps the panel focused")
	}
	h.Resize(120, 40)
	h.RequireContains("Create dir")
}

func TestOpenedPanelWhileWorktreesShow(t *testing.T) {
	b := newBrowser(&session{})
	b.w, b.h = 100, 30
	b.worktrees.visible = true
	s, _ := b.Update(openPanelMsg{Kind: panelMasks})
	if s.(browser).View() == "" {
		t.Fatal("the panel shares the column with the worktrees")
	}
}

func TestRenameDoneKeepsOtherMessages(t *testing.T) {
	if renameDone(nil) != nil {
		t.Fatal("no command, no rename")
	}
	cmd := renameDone(func() tea.Msg { return "other" })
	if cmd() != "other" {
		t.Fatal("other messages pass through")
	}
	done := renameDone(func() tea.Msg { return widgets.ModalDoneMsg{ID: "x", Index: 1} })()
	if done != (dialogDoneMsg{ID: "x", Index: 1}) {
		t.Fatalf("the answer is renamed: %v", done)
	}
}

func TestDeletingInsideAHighlightedDirectoryReloadsIt(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("down", "down") // alpha, which holds the directory sub
	h.Press("right", "down", "f8")
	h.RequireContains("Delete sub?")
	h.Press("enter")
	if _, err := os.Stat(filepath.Join(dir, "alpha", "sub")); !os.IsNotExist(err) {
		t.Fatal("the directory is deleted")
	}
	if browserOf(h).sess.currentPath() != filepath.Join(dir, "alpha") {
		t.Fatal("the highlighted directory stays")
	}
	h.RequireNotContains("📁 sub")
}
