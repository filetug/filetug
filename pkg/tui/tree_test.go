package tui

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/nav/navtest"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func treeOf(h *navtest.Harness) treeScreen { return h.Model().Menu().(treeScreen) }

func TestTreeListsSubdirectoriesWithoutHiddenOnes(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.RequireContains("▾ ..").RequireContains("📁 alpha").RequireContains("📁 beta").RequireContains("📁 Gamma")
	h.RequireNotContains(".hidden").RequireNotContains("📁 sub")
	if got := treeOf(h).Title(); got != filepath.Base(dir) {
		t.Fatalf("the panel is titled with the directory: %q", got)
	}
}

func TestTreeMovingTheCursorPreviewsTheDirectory(t *testing.T) {
	var saved []string
	dir := tree(t)
	h := open(t, dir)
	saveSelectedTreeDir = func(d string) { saved = append(saved, d) }
	h.Press("down", "down") // Gamma, alpha: uppercase sorts first
	if got := h.Model().Content().(browser).sess.currentPath(); got != filepath.Join(dir, "alpha") {
		t.Fatalf("the file list shows the highlighted directory: %q", got)
	}
	h.RequireContains("sub").Press("down")
	if len(saved) != 3 {
		t.Fatalf("the highlighted directory is remembered: %v", saved)
	}
	h.Press("up", "up", "up")
	if got := h.Model().Content().(browser).sess.currentPath(); got != dir {
		t.Fatalf("back on the root row the file list shows the root: %q", got)
	}
}

func TestTreeEnterAndLeftNavigate(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("down", "down", "enter") // Gamma, alpha
	if got := h.Model().Content().(browser).sess.treeRoot.Path(); got != filepath.Join(dir, "alpha") {
		t.Fatalf("Enter opens the directory: %q", got)
	}
	h.RequireContains("📁 sub")
	if treeOf(h).Title() != "alpha" {
		t.Fatal("title follows the root")
	}

	h.Press("left") // cursor on the root row: parent of alpha
	if got := h.Model().Content().(browser).sess.treeRoot.Path(); got != dir+"/" {
		t.Fatalf("Left goes to the parent: %q", got)
	}
	h.Press("enter") // on the root row Enter goes up as well
	if got := h.Model().Content().(browser).sess.treeRoot.Path(); got == dir {
		t.Fatalf("Enter on the root row goes up: %q", got)
	}
}

func TestTreeRightMovesFocusToTheFileList(t *testing.T) {
	h := open(t, tree(t))
	if h.Model().Zone() != nav.FocusToMenu {
		t.Fatal("the tree starts focused")
	}
	h.Press("right")
	if h.Model().Zone() != nav.FocusToContent {
		t.Fatal("Right moves to the file list")
	}
	h.Press("left")
	if h.Model().Zone() != nav.FocusToMenu {
		t.Fatal("Left in the file list moves back")
	}
}

func TestTreeUpFromTheRootRowFocusesTheBreadcrumbs(t *testing.T) {
	h := open(t, tree(t))
	h.Press("up")
	if h.Model().Zone() != nav.FocusToBreadcrumbs {
		t.Fatal("Up at the top leaves for the breadcrumbs")
	}
}

func TestTreeSearch(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Type("b")
	if treeOf(h).Title() != "Find: b" {
		t.Fatalf("the title shows the search: %q", treeOf(h).Title())
	}
	if got := h.Model().Content().(browser).sess.currentPath(); got != filepath.Join(dir, "beta") {
		t.Fatalf("the best match is shown: %q", got)
	}
	h.Type("z") // no directory contains "bz": the character is dropped again
	if treeOf(h).search != "b" {
		t.Fatalf("a pattern that matches nothing loses its last character: %q", treeOf(h).search)
	}
	h.Press("backspace")
	if treeOf(h).search != "" {
		t.Fatal("Backspace removes a character")
	}
	h.Press("backspace")
	h.Type("MM") // contains, does not start: Gamma
	if got := h.Model().Content().(browser).sess.currentPath(); got != filepath.Join(dir, "Gamma") {
		t.Fatalf("a name that contains the pattern is found: %q", got)
	}
	h.Press("esc")
	if treeOf(h).search != "" || h.Model().Content().(browser).sess.currentPath() != dir {
		t.Fatal("Esc clears the search and returns to the root")
	}
	h.Press("space")
	if treeOf(h).search != "" {
		t.Fatal("a leading space is not a search")
	}
	h.Type("a").Press("space")
	if treeOf(h).search != "a" {
		t.Fatalf("a space that matches nothing is dropped again: %q", treeOf(h).search)
	}
}

func TestTreeSearchHighlightsTheMatch(t *testing.T) {
	h := open(t, tree(t))
	h.Type("eta")
	if !strings.Contains(h.Styled(), "\x1b[") || !strings.Contains(h.View(), "b") {
		t.Fatal("the match is styled")
	}
	node := treeOf(h).nodeText("beta")
	if strings.Contains(node, "\x1b[") == false {
		t.Fatal("the matching part of the name is highlighted")
	}
	if plain := (treeScreen{search: "zz"}).nodeText("beta"); plain != "📁 beta" {
		t.Fatalf("no match, no highlight: %q", plain)
	}
}

func TestTreeGlobalShortcuts(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Type("/")
	if got := h.Model().Content().(browser).sess.treeRoot.Path(); got != "/" {
		t.Fatalf("/ goes to the root directory: %q", got)
	}
	h.RequireContains("▾ /")
	h.Type("`")
	home, _ := os.UserHomeDir()
	if got := h.Model().Content().(browser).sess.treeRoot.Path(); got != home {
		t.Fatalf("` goes home: %q", got)
	}
}

func TestTreeShowsTheReadError(t *testing.T) {
	h := open(t, tree(t))
	h.Send(goDirMsg{Path: filepath.Join(t.TempDir(), "missing")})
	h.RequireContains("no such file")
	if treeOf(h).err == nil {
		t.Fatal("the error is kept")
	}
}

func TestTreeLoadingAndStaleResults(t *testing.T) {
	sess := &session{store: fakeStore{root: url.URL{Scheme: "fake", Path: "/"}}}
	tr := newTreeScreen(sess)
	s, _ := tr.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	s, _ = s.Update(goDirMsg{Path: "/x"})
	tr = s.(treeScreen)
	if !tr.loading || !strings.Contains(tr.View(), "Loading…") {
		t.Fatal("the tree shows that the directory is being read")
	}
	sess.seq = 5
	dir := files.NewDirContext(sess.store, "/x", []os.DirEntry{files.NewDirEntry("d", true)})
	s, _ = tr.Update(dirLoadedMsg{Seq: 4, Root: true, Path: "/x", Dir: dir})
	if !s.(treeScreen).loading {
		t.Fatal("a stale result is dropped")
	}
	s, _ = tr.Update(dirLoadedMsg{Seq: 5, Root: false, Path: "/x", Dir: dir})
	if !s.(treeScreen).loading {
		t.Fatal("a result that does not replace the root is not the tree's")
	}
	s, _ = tr.Update(dirLoadedMsg{Seq: 5, Root: true, Path: "/x", Dir: dir})
	if s.(treeScreen).loading || len(s.(treeScreen).children) != 1 {
		t.Fatal("the current result is shown")
	}
	s, _ = tr.Update(dirLoadedMsg{Seq: 5, Root: true, Path: "/x", Err: errors.New("boom")})
	if got := s.(treeScreen).View(); !strings.Contains(got, "boom") {
		t.Fatalf("an error is shown: %q", got)
	}
}

func TestTreeRootText(t *testing.T) {
	store := fakeStore{root: url.URL{Scheme: "ftp", Path: "/srv/"}}
	tr := treeScreen{sess: &session{store: store}, rootPath: "/srv/"}
	if tr.rootText() != "/srv" {
		t.Fatalf("the store root: %q", tr.rootText())
	}
	tr.rootPath = "/srv/x"
	if tr.rootText() != ".." {
		t.Fatalf("below the store root: %q", tr.rootText())
	}
	tr = treeScreen{sess: &session{store: osStore()}, rootPath: "/"}
	if tr.rootText() != "/" {
		t.Fatalf("the file system root: %q", tr.rootText())
	}
}

func TestTreePanelTitle(t *testing.T) {
	sess := &session{store: fakeStore{root: url.URL{Scheme: "file", Path: "/"}}}
	tr := treeScreen{sess: sess}
	if tr.panelTitle() != "" {
		t.Fatal("no root, no title")
	}
	sess.treeRoot = files.NewDirContext(sess.store, "/", nil)
	if tr.panelTitle() != "" {
		t.Fatal("the store root has no title")
	}
	old := userHomeDir
	userHomeDir = "/home/u"
	defer func() { userHomeDir = old }()
	sess.treeRoot = files.NewDirContext(sess.store, "/home/u/", nil)
	if tr.panelTitle() != "~" {
		t.Fatal("home is ~")
	}
	sess.treeRoot = files.NewDirContext(sess.store, "/home/u/docs", nil)
	if tr.panelTitle() != "docs" {
		t.Fatal("the name of the directory")
	}
}

func TestTreeBoundaryAndHelp(t *testing.T) {
	h := open(t, tree(t))
	tr := treeOf(h)
	if tr.AtEdge(widgets.Left) || !tr.AtEdge(widgets.Right) || !tr.AtEdge(widgets.Up) || tr.AtEdge(widgets.Down) {
		t.Fatal("Right leaves always, Left never; Up and Down follow the tree")
	}
	if len(tr.ShortHelp()) != 3 || tr.Init() != nil {
		t.Fatal("help and init")
	}
}

func TestTreeIgnoresOtherMessagesAndForwardsMouse(t *testing.T) {
	h := open(t, tree(t))
	h.Click(2, 3)
	tr := treeOf(h)
	if _, cmd := tr.Update(struct{}{}); cmd != nil {
		t.Fatal("unknown messages do nothing")
	}
	if _, cmd := tr.Update(tea.KeyPressMsg{Code: tea.KeyDown}); cmd == nil {
		t.Fatal("navigation keys reach the tree")
	}
}

func TestParentOf(t *testing.T) {
	if parentOf("/a/b") != "/a/" || parentOf("/") != "/" {
		t.Fatal("parentOf")
	}
}

func TestTreeKeysBeforeAnythingIsLoaded(t *testing.T) {
	tr := newTreeScreen(&session{store: osStore()})
	s, cmd := tr.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if cmd == nil || s.(treeScreen).current() != rootID {
		t.Fatal("Left works on an empty tree")
	}
	if _, cmd = tr.Update(tea.KeyPressMsg{Code: tea.KeyRight}); cmd == nil {
		t.Fatal("Right asks the shell to focus the file list")
	}
}
