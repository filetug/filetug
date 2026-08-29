package charmui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filetug/filetug/pkg/files"
)

func TestLoadDirectoryDisablesStaleFileActionsWhileNewDirectoryLoads(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	model.focus = focusFiles
	model.entries = []os.DirEntry{
		files.NewDirEntry("old.txt", false),
		files.NewDirEntry("stale-directory", true),
	}
	model.selected = 1
	model.listOffset = 1
	model.dirRequest = 41
	directoryCanceled := false
	model.dirCancel = func() {
		directoryCanceled = true
	}

	nextPath := filepath.Join(root, "next")
	_ = model.loadDirectory(nextPath)

	if !directoryCanceled {
		t.Fatal("starting the newer directory request did not cancel the older request")
	}
	if len(model.entries) != 0 || model.selected != 0 || model.listOffset != 0 {
		t.Fatalf("loading %q retained stale file state: entries=%d selected=%d offset=%d", nextPath, len(model.entries), model.selected, model.listOffset)
	}
	if model.directoryState != "loading" {
		t.Fatalf("directory state = %q, want loading", model.directoryState)
	}
	_, _ = model.Update(directoryLoadedMsg{
		path:    root,
		request: 41,
		entries: []os.DirEntry{files.NewDirEntry("stale-directory", true)},
	})
	if len(model.entries) != 0 {
		t.Fatal("late directory result restored stale entries")
	}
	_, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if command != nil {
		t.Fatal("stale file entry started an action while the new directory was loading")
	}
	if model.currentPath != nextPath {
		t.Fatalf("stale file action changed current path to %q, want %q", model.currentPath, nextPath)
	}
}

func TestLongTitleStaysOneLineWithin80By24Layout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	model.currentPath = filepath.Join(root, strings.Repeat("directory-with-a-long-name-", 6)+"界🙂")

	title := model.titleView()
	plainTitle := ansi.Strip(title)
	if strings.Contains(plainTitle, "\n") {
		t.Fatalf("title wrapped into multiple lines: %q", plainTitle)
	}
	if width := ansi.StringWidth(title); width > model.width {
		t.Fatalf("title width = %d, exceeds terminal width %d: %q", width, model.width, plainTitle)
	}
	if !strings.Contains(plainTitle, "…") {
		t.Fatalf("long title was not ellipsized: %q", plainTitle)
	}

	content := ansi.Strip(model.View().Content)
	if lines := strings.Count(content, "\n") + 1; lines > model.height {
		t.Fatalf("view rendered %d lines in a %d-line terminal", lines, model.height)
	}
}

func TestPanelsKeepEqualOuterHeightWithSparseContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	layout := model.panelLayout()
	panels := []struct {
		name   string
		height int
	}{
		{name: "directories", height: lipgloss.Height(model.panel("Directories", model.tree.View(), focusTree, layout.treeOuter))},
		{name: "files", height: lipgloss.Height(model.panel("Files", model.filesView(), focusFiles, layout.filesOuter))},
		{name: "preview", height: lipgloss.Height(model.panel("Preview", model.preview.View(), focusPreview, layout.previewOuter))},
	}
	want := model.height - 2 // One application title row and one footer row.
	for _, panel := range panels {
		if panel.height != want {
			t.Errorf("%s panel height = %d, want %d", panel.name, panel.height, want)
		}
	}
}
