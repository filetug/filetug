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
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/nav/navtest"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func browserOf(h *navtest.Harness) browser { return h.Model().Content().(browser) }

func TestBrowserShowsTheStartDirectory(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.RequireContains("README.md").RequireContains("a.go").RequireContains("2 items")
	h.RequireNotContains("Gamma      ┃") // directories are in the tree, not in the list
	crumbs := h.Model().Breadcrumbs()
	if crumbs[0].Title != "FileTug" || crumbs[len(crumbs)-1].Title != filepath.Base(dir) {
		t.Fatalf("the breadcrumbs show the directory: %v", crumbs)
	}
}

func TestBrowserPreviewFollowsTheCursor(t *testing.T) {
	var names []string
	h := open(t, tree(t))
	saveCurrentFileName = func(n string) { names = append(names, n) }
	h.Press("right", "down", "down") // ".." then README.md then a.go
	h.RequireContains("package main")
	if b := browserOf(h); b.preview.Title() != "a.go" || b.sess.entryName != "a.go" {
		t.Fatalf("the preview and the remembered file follow the cursor: %q %q", b.preview.Title(), b.sess.entryName)
	}
	if len(names) != 2 {
		t.Fatalf("file names are remembered: %v", names)
	}
	h.Press("up", "up")
	h.RequireContains("Modified")
}

func TestBrowserEnterOpensDirectoriesOnly(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("down", "down", "enter") // open alpha via the tree
	h.Press("right", "enter")        // ".." row of the file list: the parent
	if got := browserOf(h).sess.treeRoot.Path(); got != strings.TrimSuffix(dir, "/") {
		t.Fatalf("Enter on .. goes to the parent: %q", got)
	}
	h.Press("down", "down", "enter") // README.md: nothing happens
	if got := browserOf(h).sess.treeRoot.Path(); got != dir {
		t.Fatalf("Enter on a file does not navigate: %q", got)
	}
}

func TestBrowserSelectsTheRememberedFile(t *testing.T) {
	dir := tree(t)
	isolateState(t)
	sess := &session{store: osStore(), entryName: "a.go"}
	b := newBrowser(sess)
	b.startMsg = goDirMsg{Path: dir}
	h := navtest.New(t, nav.Page{Title: "t", Menu: newTreeScreen(sess), Content: b}, navtest.WithSize(140, 30))
	h.RequireContains("Size")
	if sess.entryName != "a.go" || browserOf(h).preview.Title() != "a.go" {
		t.Fatalf("the remembered file is selected: %q", browserOf(h).preview.Title())
	}
}

func TestBrowserSpaceMarksEntries(t *testing.T) {
	h := open(t, tree(t))
	h.Press("right", "down", "space")
	h.RequireContains("✓📄 README.md").RequireContains("1 selected")
	h.Press("space")
	h.RequireNotContains("✓").RequireNotContains("selected")
	h.Press("up", "space") // the parent row cannot be marked
	h.RequireNotContains("✓")
}

func TestBrowserGlobalShortcutsFromTheFileList(t *testing.T) {
	h := open(t, tree(t))
	h.Press("right").Type("/")
	if browserOf(h).sess.treeRoot.Path() != "/" {
		t.Fatal("/ goes to the root")
	}
	h.Type("`")
	home, _ := os.UserHomeDir()
	if browserOf(h).sess.treeRoot.Path() != home {
		t.Fatal("` goes home")
	}
}

func TestBrowserDropsStaleResults(t *testing.T) {
	sess := &session{store: osStore(), seq: 2, previewSeq: 4}
	b := newBrowser(sess)
	s, _ := b.Update(dirLoadedMsg{Seq: 1, Path: "/x", Err: errors.New("late")})
	if s.(browser).files.err != nil {
		t.Fatal("a stale directory is dropped")
	}
	s, _ = b.Update(previewMsg{Seq: 3, Title: "late"})
	if s.(browser).preview.Title() == "late" {
		t.Fatal("a stale preview is dropped")
	}
	s, _ = b.Update(previewMsg{Seq: 4, Title: "current"})
	if s.(browser).preview.Title() != "current" {
		t.Fatal("the current preview is shown")
	}
}

func TestBrowserShowsReadErrors(t *testing.T) {
	sess := &session{store: osStore(), seq: 1, treeRoot: files.NewDirContext(nil, "/x", nil)}
	s0, _ := newBrowser(sess).Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	s, _ := s0.Update(dirLoadedMsg{Seq: 1, Path: "/x/y", Err: errors.New("denied")})
	got := s.(browser)
	if got.files.err == nil || got.preview.Text() != "denied" || !strings.Contains(got.View(), "denied") {
		t.Fatalf("the error is shown in both panes: %q", got.preview.Text())
	}
}

func TestBrowserSelectionOfAnEmptyListClearsThePreview(t *testing.T) {
	sess := &session{store: osStore()}
	b := newBrowser(sess)
	b.files.SetDir(files.NewDirContext(nil, "/", nil), false, "")
	s, cmd := b.Update(grid.SelectionChangedMsg{ID: filesID})
	if cmd != nil || s.(browser).preview.Text() != "" {
		t.Fatal("nothing to preview")
	}
	if _, cmd := b.Update(grid.SelectionChangedMsg{ID: "other"}); cmd != nil {
		t.Fatal("other grids are not the file list")
	}
}

func TestBrowserActivatedIgnoresForeignRows(t *testing.T) {
	b := newBrowser(&session{})
	dirRow := grid.Row{Ref: rowRef{Entry: files.NewEntryWithDirPath(files.NewDirEntry("d", true), "/p"), IsDir: true}}
	if cmd := b.activated(grid.RowActivatedMsg{ID: "other", Row: dirRow}); cmd != nil {
		t.Fatal("other grids")
	}
	if cmd := b.activated(grid.RowActivatedMsg{ID: filesID, Row: grid.Row{}}); cmd != nil {
		t.Fatal("rows without a reference")
	}
	if cmd := b.activated(grid.RowActivatedMsg{ID: filesID, Row: dirRow}); cmd == nil {
		t.Fatal("a directory row opens")
	}
}

func TestBrowserBreadcrumbClicksNavigate(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("down", "down", "enter") // into alpha
	crumbs := h.Model().Breadcrumbs()
	last := crumbs[len(crumbs)-1].Title
	if last != "alpha" {
		t.Fatalf("crumbs: %v", crumbs)
	}
	h.Send(widgets.CrumbSelectedMsg{ID: crumbsID, Index: len(crumbs) - 2})
	if got := browserOf(h).sess.treeRoot.Path(); got != dir {
		t.Fatalf("the parent crumb opens the parent: %q", got)
	}
	before := browserOf(h).sess.seq
	h.Send(widgets.CrumbSelectedMsg{ID: crumbsID, Index: 0})
	h.Send(widgets.CrumbSelectedMsg{ID: "other", Index: 1})
	h.Send(widgets.CrumbSelectedMsg{ID: crumbsID, Index: 99})
	if browserOf(h).sess.seq != before {
		t.Fatal("the first crumb, foreign ids and unknown positions do nothing")
	}
}

func TestCrumbsFor(t *testing.T) {
	store := fakeStore{root: url.URL{Scheme: "fake", Path: "/srv"}}
	crumbs, paths := crumbsFor(nil, "/x")
	if len(crumbs) != 1 || paths != nil {
		t.Fatal("no store, no path")
	}
	crumbs, paths = crumbsFor(store, "")
	if len(crumbs) != 2 || len(paths) != 1 || paths[0] != "/srv" || crumbs[1].Title != "fake" {
		t.Fatalf("only the store: %v %v", crumbs, paths)
	}
	crumbs, _ = crumbsFor(store, "/srv")
	if len(crumbs) != 2 {
		t.Fatal("the store root itself")
	}
	crumbs, paths = crumbsFor(store, "/srv/a//b/")
	titles := []string{}
	for _, c := range crumbs {
		titles = append(titles, c.Title)
	}
	if strings.Join(titles, ",") != "FileTug,fake,a,{EMPTY PATH ITEM},b" || paths[2] != "/srv/a" || paths[3] != "/srv/a/b" {
		t.Fatalf("%v %v", titles, paths)
	}
	_, paths = crumbsFor(fakeStore{root: url.URL{Scheme: "file"}}, "/a")
	if paths[0] != "/" || paths[1] != "/a" {
		t.Fatalf("an empty store path is the root: %v", paths)
	}
}

func TestBrowserResize(t *testing.T) {
	h := open(t, tree(t))
	h.Press("alt+=")
	if w := browserOf(h).weights; w != [2]int{13, 6} {
		t.Fatalf("the file list grows: %v", w)
	}
	h.Press("alt+-", "alt+-")
	if w := browserOf(h).weights; w != [2]int{11, 8} {
		t.Fatalf("and shrinks: %v", w)
	}
	h.Press("right", "right", "alt+=") // focus the preview: it grows
	if w := browserOf(h).weights; w != [2]int{10, 9} {
		t.Fatalf("the focused pane grows: %v", w)
	}
	b := browserOf(h)
	for i := 0; i < 20; i++ {
		b = b.resize(resizeMsg{Delta: -1})
	}
	if b.weights[0] < 1 || b.weights[1] < 1 {
		t.Fatalf("a pane never disappears: %v", b.weights)
	}
	h.Press("alt+0")
	if browserOf(h).weights != defaultProportions {
		t.Fatal("Alt+0 restores the sizes")
	}
}

func TestBrowserFocusAndBoundaries(t *testing.T) {
	h := open(t, tree(t))
	h.Press("right")
	b := browserOf(h)
	if !b.AtEdge(widgets.Left) || b.AtEdge(widgets.Right) || !b.AtEdge(widgets.Up) {
		t.Fatal("the file list hands Left to the tree and keeps Right")
	}
	h.Press("right")
	b = browserOf(h)
	if b.focus != panePreview || b.AtEdge(widgets.Left) || !b.AtEdge(widgets.Up) {
		t.Fatal("the preview is focused and keeps Left")
	}
	h.Press("left")
	if browserOf(h).focus != paneFiles {
		t.Fatal("Left goes back to the file list")
	}
	if !b.Borderless() || b.Init() == nil || len(b.ShortHelp()) != 2 {
		t.Fatal("static properties")
	}
}

func TestBrowserMouse(t *testing.T) {
	h := open(t, tree(t))
	h.Click(100, 5) // inside the preview: x is relative to the panel
	if browserOf(h).focus != panePreview {
		t.Fatal("a click focuses the preview")
	}
	h.Click(45, 5)
	if browserOf(h).focus != paneFiles {
		t.Fatal("a click focuses the file list")
	}
	h.Press("right", "right")
	h.Wheel(120, 5, false).Wheel(120, 5, true)
	h.Click(45, 5)
	h.Press("down")
	h.Wheel(45, 5, false)
	h.RequireContains("a.go")
	h.Wheel(45, 5, true)
	b := browserOf(h)
	if _, cmd := b.mouse(tea.MouseMotionMsg{X: 1}); cmd != nil {
		t.Fatal("motion is ignored")
	}
}

func TestBrowserEmptyAndTinyViews(t *testing.T) {
	b := newBrowser(&session{})
	if b.View() != "" {
		t.Fatal("no size, no view")
	}
	s, _ := b.Update(tea.WindowSizeMsg{Width: 3, Height: 3})
	if s.(browser).View() == "" {
		t.Fatal("a tiny view still draws")
	}
	if got := (browser{sess: &session{current: files.NewDirContext(nil, "/", nil)}}).filesTitle(); got != "/" {
		t.Fatalf("root title %q", got)
	}
	if got := (browser{sess: &session{}}).filesTitle(); got != "" {
		t.Fatalf("no title without a directory: %q", got)
	}
}

func TestGitStatusDecoratesTheTreeAndTheList(t *testing.T) {
	dir := tree(t)
	fakeGit(t, dir, map[string]*gitutils.RepoStatus{
		dir:                                changes("main", 0, 0, 0),
		filepath.Join(dir, "alpha"):        changes("main", 2, 7, 1),
		filepath.Join(dir, "a.go"):         changes("main", 1, 3, 0),
		filepath.Join(dir, "README.md"):    changes("main", 0, 0, 0),
		filepath.Join(dir, "alpha", "sub"): nil,
	})
	h := open(t, dir)
	h.RequireContains("alpha ┆main┆ƒ2+7-1").RequireContains("a.go ┆main┆ƒ1+3").RequireContains(".. ┆main±0")
	h.RequireNotContains("README.md ┆")
	h.RequireNotContains("beta ┆")

	// Moving the cursor starts a new list request but keeps the tree's statuses.
	h.Press("down", "down")
	h.RequireContains("alpha ┆main┆ƒ2+7-1")
}

func TestGitStatusIsOnlyReadOnTheLocalFileSystem(t *testing.T) {
	b := newBrowser(&session{store: fakeStore{root: url.URL{Scheme: "ftp"}}})
	if b.gitCmds(dirLoadedMsg{}) != nil {
		t.Fatal("no repositories on an FTP server")
	}
}

func TestGitStatusForADirectoryShownFromTheTree(t *testing.T) {
	dir := tree(t)
	fakeGit(t, dir, map[string]*gitutils.RepoStatus{filepath.Join(dir, "alpha", "x"): changes("main", 1, 1, 0)})
	write(t, filepath.Join(dir, "alpha", "x"), "x")
	h := open(t, dir)
	h.Press("down", "down") // alpha: Gamma, alpha
	h.RequireContains("x ┆main┆ƒ1+1")
}

func TestStaleGitStatusIsDropped(t *testing.T) {
	sess := &session{store: osStore(), seq: 3, rootSeq: 2}
	b := newBrowser(sess)
	ch := make(chan gitStatusMsg)
	for _, msg := range []gitStatusMsg{
		{Seq: 1, Scope: gitFiles, Path: "/x", Text: "t", stream: ch},
		{Seq: 1, Scope: gitTree, Path: "/x", Text: "t", stream: ch},
	} {
		if _, cmd := b.Update(msg); cmd != nil {
			t.Fatalf("a stale status ends its stream: %+v", msg)
		}
	}
	close(ch)
	if _, cmd := b.Update(gitStatusMsg{Seq: 3, Scope: gitFiles, Path: "/x", Text: "t", stream: ch}); cmd == nil {
		t.Fatal("a current status asks for the next one")
	}
	tr := newTreeScreen(sess)
	s, _ := tr.Update(gitStatusMsg{Seq: 9, Scope: gitTree, Path: "/x", Text: "t"})
	s, _ = s.Update(gitStatusMsg{Seq: 2, Scope: gitFiles, Path: "/x", Text: "t"})
	if len(s.(treeScreen).git) != 0 {
		t.Fatal("the tree shows only current statuses of its own list")
	}
}
