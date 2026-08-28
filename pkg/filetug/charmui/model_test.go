package charmui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/filetug/filetug/pkg/files"
)

type testStore struct{}

func (testStore) ReadDir(context.Context, string) ([]os.DirEntry, error) {
	return nil, nil
}

func TestModelJourneyTreeToFileToMarkdownPreview(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.readFile = func(_ context.Context, path string, _ int64) (string, error) {
		if path != filepath.Join(root, "docs", "readme.md") {
			t.Fatalf("unexpected preview path: %s", path)
		}
		return "# Filetug MVP\n\nTree navigation works.", nil
	}

	initial := model.Init()
	if initial == nil {
		t.Fatal("Init did not request the root directory")
	}
	_, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_, _ = model.Update(directoryLoadedMsg{
		path:    root,
		request: model.dirRequest,
		entries: []os.DirEntry{files.NewDirEntry("docs", true)},
	})

	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, command := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if command == nil || model.currentPath != filepath.Join(root, "docs") {
		t.Fatalf("tree selection did not request docs: %q", model.currentPath)
	}
	_, command = model.Update(directoryLoadedMsg{
		path:    model.currentPath,
		request: model.dirRequest,
		entries: []os.DirEntry{files.NewDirEntry("readme.md", false)},
	})
	if command == nil {
		t.Fatal("directory selection did not request a preview")
	}
	previewMessage := command()
	_, _ = model.Update(previewMessage)

	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Filetug MVP") {
		t.Fatalf("markdown preview is missing from view: %q", view)
	}
	if model.currentPath != filepath.Join(root, "docs") {
		t.Fatalf("current directory = %q", model.currentPath)
	}
}

func TestModelRejectsStalePreview(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.entries = []os.DirEntry{
		files.NewDirEntry("a.txt", false),
		files.NewDirEntry("b.txt", false),
	}
	model.setSize(80, 20)
	model.readFile = func(_ context.Context, path string, _ int64) (string, error) {
		return "preview " + filepath.Base(path), nil
	}

	model.selected = 0
	first := model.loadPreviewForSelection()
	model.selected = 1
	second := model.loadPreviewForSelection()

	_, _ = model.Update(first())
	if strings.Contains(model.preview.View(), "a.txt") {
		t.Fatal("stale preview overwrote the newer selection")
	}
	_, _ = model.Update(second())
	if !strings.Contains(model.preview.View(), "b.txt") {
		t.Fatal("newest preview was not accepted")
	}
}

func TestFilesViewVirtualizesRowsAndLayoutFitsWindow(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 10)
	entries := make([]os.DirEntry, 100)
	for index := range entries {
		entries[index] = files.NewDirEntry("entry-"+strings.Repeat("x", index%3), false)
	}
	model.entries = entries
	view := model.filesView()
	lines := strings.Count(view, "\n") + 1
	if lines > model.listHeight() {
		t.Fatalf("rendered %d rows with a visible window of %d", lines, model.listHeight())
	}
	layout := model.panelLayout()
	if layout.treeOuter+layout.filesOuter+layout.previewOuter+2 != model.width {
		t.Fatalf("panel outer widths exceed window: %+v width=%d", layout, model.width)
	}
}
