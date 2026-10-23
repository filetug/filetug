package charmui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
)

func TestNewModelRejectsNilStore(t *testing.T) {
	t.Parallel()
	_, err := NewModel(t.TempDir(), nil)
	if err == nil {
		t.Fatal("nil directory store was accepted")
	}
}

func TestNewModelReportsUnresolvablePath(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err = os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	if err = os.RemoveAll(directory); err != nil {
		t.Fatal(err)
	}
	_, err = NewModel("relative", testStore{})
	if err == nil {
		t.Fatal("expected an error resolving a relative path without a working directory")
	}
}

func TestRunLocalReportsProgramAndPathErrors(t *testing.T) {
	root := t.TempDir()
	originalRun := runProgram
	t.Cleanup(func() {
		runProgram = originalRun
	})

	runProgram = func(tea.Model) error {
		return errors.New("terminal unavailable")
	}
	err := RunLocal(root)
	if err == nil || !strings.Contains(err.Error(), "run Bubble Tea UI") {
		t.Fatalf("RunLocal error = %v", err)
	}

	runProgram = func(tea.Model) error {
		return nil
	}
	if err = RunLocal(root); err != nil {
		t.Fatalf("RunLocal success = %v", err)
	}

	// Execute the real program once so its body stays covered. Init quits
	// immediately when a terminal exists; otherwise Run returns an error.
	if programErr := originalRun(quitModel{}); programErr != nil {
		t.Logf("default program runner: %v", programErr)
	}

	original, wdErr := os.Getwd()
	if wdErr != nil {
		t.Fatal(wdErr)
	}
	if err = os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
	if err = os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err = RunLocal(""); err == nil {
		t.Fatal("empty path should fail when the working directory is gone")
	}
}

// quitModel exits on the first update so a real Bubble Tea program cannot
// block the coverage test when a terminal is available.
type quitModel struct{}

func (quitModel) Init() tea.Cmd { return tea.Quit }

func (quitModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return quitModel{}, tea.Quit }

func (quitModel) View() tea.View { return tea.NewView("") }

type scriptedStore struct {
	entries []os.DirEntry
	err     error
}

func (s scriptedStore) ReadDir(context.Context, string) ([]os.DirEntry, error) {
	return s.entries, s.err
}

func TestDirectoryLoadCommandAndAcceptPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, scriptedStore{err: errors.New("unreadable")})
	if err != nil {
		t.Fatal(err)
	}
	command := model.loadDirectory(root)
	message := command()
	loaded, ok := message.(directoryLoadedMsg)
	if !ok || loaded.err == nil {
		t.Fatalf("directory command = %#v", message)
	}
	_, follow := model.Update(loaded)
	if follow != nil {
		t.Fatal("directory error should not request a preview")
	}
	if model.directoryState != "error" || !strings.Contains(model.previewState, "unreadable") {
		t.Fatalf("directory error state = %q preview = %q", model.directoryState, model.previewState)
	}

	empty := directoryLoadedMsg{path: model.currentPath, request: model.dirRequest}
	_, follow = model.Update(empty)
	if follow != nil || model.directoryState != "empty" {
		t.Fatalf("empty directory state = %q command = %v", model.directoryState, follow)
	}

	entries := []os.DirEntry{
		files.NewDirEntry("b.txt", false),
		files.NewDirEntry("a", true),
		files.NewDirEntry("A.txt", false),
	}
	model.acceptDirectory(directoryLoadedMsg{path: model.currentPath, request: model.dirRequest, entries: entries})
	if len(model.entries) != 3 || model.entries[0].Name() != "a" || model.entries[1].Name() != "A.txt" {
		t.Fatalf("sorted entries = %s %s", model.entries[0].Name(), model.entries[1].Name())
	}
}

func TestFileKeysMoveSelectionAndOpenDirectories(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	entries := make([]os.DirEntry, 30)
	for index := range entries {
		name := "file-" + string(rune('a'+(index%26)))
		entries[index] = files.NewDirEntry(name, false)
	}
	entries[2] = files.NewDirEntry("nested", true)
	model.entries = entries
	model.directoryState = "ready"
	model.readFile = func(context.Context, string, int64) (string, error) {
		return "", errors.New("preview failed")
	}

	_, command := model.updateFiles("down")
	if model.selected != 1 || command == nil {
		t.Fatalf("down selected = %d command = %v", model.selected, command)
	}
	preview := command()
	failed, ok := preview.(previewLoadedMsg)
	if !ok || failed.err == nil {
		t.Fatalf("preview command = %#v", preview)
	}
	model.acceptPreview(failed)
	if !strings.Contains(model.previewState, "preview failed") {
		t.Fatalf("preview state = %q", model.previewState)
	}

	_, command = model.updateFiles("up")
	if model.selected != 0 || command == nil {
		t.Fatalf("up selected = %d", model.selected)
	}
	_, command = model.updateFiles("up")
	if model.selected != 0 || command != nil {
		t.Fatalf("up at top selected = %d command = %v", model.selected, command)
	}

	model.selected = len(entries) - 1
	_, command = model.updateFiles("down")
	if model.selected != len(entries)-1 || command != nil {
		t.Fatalf("down at end selected = %d command = %v", model.selected, command)
	}

	model.selected = 0
	_, _ = model.updateFiles("pgdown")
	if model.selected == 0 {
		t.Fatal("page down left the selection on the first row")
	}
	_, _ = model.updateFiles("pgdown")
	if model.selected != len(entries)-1 {
		t.Fatalf("second page down selected = %d, want last row", model.selected)
	}
	_, _ = model.updateFiles("pgup")
	if model.selected == len(entries)-1 {
		t.Fatal("page up left the selection on the last row")
	}

	model.selected = 2
	_, command = model.updateFiles("enter")
	want := filepath.Join(root, "nested")
	if command == nil || model.currentPath != want {
		t.Fatalf("enter directory path = %q command = %v", model.currentPath, command)
	}

	model.currentPath = root
	model.entries = entries
	model.selected = 0
	_, command = model.updateFiles("right")
	if command != nil || model.currentPath != root {
		t.Fatalf("enter on a file changed path to %q command = %v", model.currentPath, command)
	}

	outside := filepath.Dir(root)
	model.currentPath = outside
	_, command = model.updateFiles("left")
	if command != nil || model.currentPath != outside {
		t.Fatalf("parent outside the root was opened: %q command = %v", model.currentPath, command)
	}
}

func TestFocusKeysQuitPreviewAndUnknownFocus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	canceledDirectory := false
	canceledPreview := false
	model.dirCancel = func() { canceledDirectory = true }
	model.previewCancel = func() { canceledPreview = true }

	_, command := model.Update(tea.KeyPressMsg{Code: 'q'})
	if command == nil || !canceledDirectory || !canceledPreview {
		t.Fatalf("quit command = %v directoryCanceled = %v previewCanceled = %v", command, canceledDirectory, canceledPreview)
	}

	model.focus = focusPreview
	_, command = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if command == nil && model.focus != focusPreview {
		t.Fatal("preview key changed focus")
	}

	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if model.focus != focusFiles {
		t.Fatalf("shift+tab from preview focus = %d, want files", model.focus)
	}

	model.focus = 9
	_, command = model.Update(tea.KeyPressMsg{Code: 'x'})
	if command != nil {
		t.Fatal("unknown focus handled a key")
	}
}

func TestSelectedPathAndVisibleWindow(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	model.setSize(80, 24)
	model.entries = []os.DirEntry{files.NewDirEntry("a.txt", false)}
	model.selected = -1
	if model.selectedPath() != "" {
		t.Fatalf("selected path = %q, want empty", model.selectedPath())
	}
	command := model.loadPreviewForSelection()
	if command != nil || model.previewState != "No file selected" {
		t.Fatalf("empty selection preview = %q command = %v", model.previewState, command)
	}

	model.entries = make([]os.DirEntry, 40)
	for index := range model.entries {
		model.entries[index] = files.NewDirEntry("row", false)
	}
	model.selected = 0
	model.listOffset = 5
	model.ensureSelectedVisible()
	if model.listOffset != 0 {
		t.Fatalf("offset = %d, want the selected row scrolled into view", model.listOffset)
	}
	model.selected = 30
	model.listOffset = 0
	model.ensureSelectedVisible()
	if model.listOffset == 0 {
		t.Fatal("selection below the window was not scrolled")
	}
}

func TestFileListStatesAndCompactView(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	model, err := NewModel(root, testStore{})
	if err != nil {
		t.Fatal(err)
	}
	loading := model.View()
	if !strings.Contains(loading.Content, "Loading Filetug") || !loading.AltScreen {
		t.Fatalf("startup view = %#v", loading)
	}

	model.directoryState = "empty"
	if !strings.Contains(model.filesView(), "empty") {
		t.Fatalf("empty files view = %q", model.filesView())
	}
	model.directoryState = "error"
	model.err = "denied"
	if !strings.Contains(model.filesView(), "denied") {
		t.Fatalf("error files view = %q", model.filesView())
	}
	model.directoryState = "ready"
	model.entries = nil
	if !strings.Contains(model.filesView(), "empty") {
		t.Fatalf("ready empty files view = %q", model.filesView())
	}

	model.setSize(80, 24)
	model.entries = []os.DirEntry{
		files.NewDirEntry("docs", true),
		files.NewDirEntry("notes.txt", false),
	}
	model.selected = 1
	view := model.filesView()
	if !strings.Contains(view, "docs/") || !strings.Contains(view, "notes.txt") {
		t.Fatalf("files view = %q", view)
	}

	model.setSize(40, 24)
	compact := model.View()
	if !strings.Contains(compact.Content, "compact") || !compact.AltScreen {
		t.Fatalf("compact view = %#v", compact)
	}
}

func TestReadLocalFileAndMarkdownRenderError(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	filePath := filepath.Join(directory, "note.txt")
	content := "hello preview"
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readLocalFile(context.Background(), filePath, int64(len(content)))
	if err != nil || got != content {
		t.Fatalf("readLocalFile = %q, %v", got, err)
	}
	truncated, err := readLocalFile(context.Background(), filePath, 4)
	if err != nil || !strings.Contains(truncated, "[preview truncated]") {
		t.Fatalf("truncated preview = %q, %v", truncated, err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = readLocalFile(canceled, filePath, 10)
	if err == nil {
		t.Fatal("canceled read succeeded")
	}
	_, err = readLocalFile(context.Background(), filepath.Join(directory, "missing.txt"), 10)
	if err == nil {
		t.Fatal("missing file succeeded")
	}
	_, err = readLocalFile(context.Background(), directory, 10)
	if err == nil {
		t.Fatal("reading a directory succeeded")
	}

	plain, err := renderPreview(filePath, content)
	if err != nil || plain != content {
		t.Fatalf("plain preview = %q, %v", plain, err)
	}

	missingStyle := filepath.Join(directory, "missing-style.json")
	_, err = renderPreviewWithStyle(filepath.Join(directory, "note.md"), "# Title", missingStyle)
	if err == nil {
		t.Fatal("markdown render with a missing style succeeded")
	}
	_, err = renderPreviewWithStyle(filepath.Join(directory, "note.markdown"), "# Title", missingStyle)
	if err == nil {
		t.Fatal("markdown extension render with a missing style succeeded")
	}
}

func TestPathWithinRootRejectsMixedPaths(t *testing.T) {
	t.Parallel()
	if isPathWithinRoot("relative", "/absolute") {
		t.Fatal("mixed relative and absolute paths were treated as inside the root")
	}
	if !isPathWithinRoot("/tmp/project", "/tmp/project") {
		t.Fatal("a path equal to the root was rejected")
	}
}
