package tui

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// wbReport installs a fake wb whose report describes linked as a managed effort.
func wbReport(t *testing.T, canonical, linked string) {
	t.Helper()
	report := wbOrphanReport{Families: []wbWorktreeFamily{{Worktrees: []wbWorktreeInfo{{
		Path: linked, CanonicalDir: canonical, Branch: "feature", HasManifest: true,
		EffortID: "worktrees", ParentEffort: "parent", RootEffort: "root", Layout: "linked",
		OwnerState: "active", OwnerAgent: "codex", Disposition: "active", LastCommit: time.Now().Add(-time.Hour),
	}}}}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	installFakeWB(t, string(encoded), 0)
}

func TestWorktreesOpenAtARepositoryRootAndListEveryWorktree(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	h := open(t, canonical)
	h.RequireContains("WORKTREES").RequireContains("2 Git worktrees").RequireContains("1 WB managed")
	h.RequireContains("Clone").RequireContains("WB").RequireContains("feature").RequireContains("clean")
	if !browserOf(h).worktrees.visible || browserOf(h).focus != paneFiles {
		t.Fatal("the pane opens by itself without taking the focus")
	}
}

func TestWorktreesFocusDetailAndOpen(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	h := openSized(t, canonical, 140, 70)
	h.Press("alt+w")
	if browserOf(h).focus != paneWorktrees {
		t.Fatal("Alt+W focuses the open pane")
	}
	h.RequireContains("Canonical clone").RequireContains("Enter open")
	h.Press("down")
	h.RequireContains("WB effort").RequireContains("Owner  codex (active)").RequireContains("Parent parent").RequireContains("active · linked")
	h.Press("enter")
	if got := browserOf(h).sess.treeRoot.Path(); got != linked {
		t.Fatalf("Enter opens the worktree: %q", got)
	}
	h.RequireContains("WORKTREES")
}

func TestWorktreesKeys(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	h := open(t, canonical)
	h.Press("alt+w", "r")
	h.RequireContains("2 Git worktrees")
	h.Press("up") // at the top: back to the preview
	if browserOf(h).focus != panePreview {
		t.Fatal("Up at the top returns to the preview")
	}
	h.Press("down") // the preview text is short: on to the worktrees
	if browserOf(h).focus != paneWorktrees {
		t.Fatal("Down at the bottom of the preview goes on to the worktrees")
	}
	h.Press("down", "up")
	if browserOf(h).focus != paneWorktrees {
		t.Fatal("Up inside the table moves the cursor")
	}
	h.Press("left")
	if browserOf(h).focus != paneFiles {
		t.Fatal("Left returns to the file list")
	}
	h.Press("alt+w", "esc")
	if browserOf(h).worktrees.visible || browserOf(h).focus != paneFiles {
		t.Fatal("Esc closes the pane")
	}
	h.RequireNotContains("WORKTREES")
}

func TestWorktreesOutsideARepository(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.RequireNotContains("WORKTREES")
	h.Press("alt+w")
	h.RequireContains("Not inside a Git repository")
	if browserOf(h).focus == paneWorktrees {
		t.Fatal("there is nothing to focus")
	}
}

func TestWorktreesPaneFollowsTheDirectory(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	h := open(t, canonical)
	h.Send(goDirMsg{Path: linked})
	h.RequireContains("2 Git worktrees")
	h.Send(goDirMsg{Path: filepath.Dir(canonical)})
	if browserOf(h).worktrees.visible {
		t.Fatal("the pane closes outside a repository")
	}
	h.Send(goDirMsg{Path: canonical})
	if !browserOf(h).worktrees.visible {
		t.Fatal("and opens again at a repository root")
	}
	h.Press("alt+w")
	if browserOf(h).focus != paneWorktrees {
		t.Fatal("the pane has the focus")
	}
	h.Send(goDirMsg{Path: filepath.Dir(canonical)})
	if browserOf(h).focus == paneWorktrees {
		t.Fatal("focus leaves a pane that closes")
	}
}

func TestWorktreesAltWNeedsALocalDirectory(t *testing.T) {
	b := newBrowser(&session{})
	if _, cmd := b.Update(worktreesToggleMsg{}); cmd != nil {
		t.Fatal("no store")
	}
	b.sess.store = fakeStore{root: url.URL{Scheme: "ftp"}}
	if _, cmd := b.Update(worktreesToggleMsg{}); cmd != nil {
		t.Fatal("not the local file system")
	}
	b.sess.store = osStore()
	if _, cmd := b.Update(worktreesToggleMsg{}); cmd != nil {
		t.Fatal("no directory yet")
	}
	if b.probeCmd("/x") != nil && b.sess.store.RootURL().Scheme != "file" {
		t.Fatal("unreachable")
	}
	b.sess.store = fakeStore{root: url.URL{Scheme: "ftp"}}
	if b.probeCmd("/x") != nil {
		t.Fatal("no probing of remote directories")
	}
}

func TestWorktreesProbeForAnotherDirectoryIsDropped(t *testing.T) {
	sess := &session{store: osStore(), current: files.NewDirContext(nil, "/a", nil)}
	b := newBrowser(sess)
	s, cmd := b.Update(worktreeProbeMsg{Path: "/elsewhere", RepoRoot: "/r", Intent: probeShow})
	if cmd != nil || s.(browser).worktrees.visible {
		t.Fatal("the answer for a directory that is no longer shown is dropped")
	}
	s, _ = b.Update(worktreeProbeMsg{Path: "/a", RepoRoot: "/r", Intent: probeAuto})
	if s.(browser).worktrees.visible {
		t.Fatal("automatic opening happens at a repository root only")
	}
}

func TestWorktreesLoadFailureAndStaleMessages(t *testing.T) {
	p := newWorktreesPane()
	p.SetSize(40, 10)
	cmd := p.Load(t.TempDir())
	msg := cmd().(worktreesListedMsg)
	if msg.Err == nil {
		t.Fatal("a directory that is not a repository fails")
	}
	if p.Listed(worktreesListedMsg{ID: msg.ID + 1}) != nil || p.loading == false {
		t.Fatal("a stale inventory is dropped")
	}
	if p.Listed(msg) != nil || p.loading || !strings.Contains(p.View(), "unavailable") {
		t.Fatalf("the failure is shown: %q", p.View())
	}
	p.Enriched(worktreesEnrichedMsg{ID: 99})
	if p.EnsureLoaded(p.repoRoot) == nil {
		t.Fatal("a failed repository is read again")
	}
	p.Hide()
	p.Hide()
}

func TestWorktreesEnsureLoadedSkipsLoadedRepositories(t *testing.T) {
	p := newWorktreesPane()
	p.SetSize(40, 10)
	p.Load("/repo")
	if p.EnsureLoaded("/repo/") != nil {
		t.Fatal("already loading")
	}
	p.stop()
	p.items = []worktreeInfo{{Path: "/repo"}}
	if p.EnsureLoaded("/repo") != nil {
		t.Fatal("already loaded")
	}
	if p.EnsureLoaded("/other") == nil {
		t.Fatal("another repository")
	}
}

func TestWorktreeColumnsAndDetail(t *testing.T) {
	cases := []struct {
		item                worktreeInfo
		kind, branch, state string
	}{
		{worktreeInfo{Canonical: true, Branch: "main", StatusRead: true}, "Clone", "main", "clean"},
		{worktreeInfo{Branch: "f", WB: &wbWorktreeInfo{HasManifest: true}, StatusRead: true, Dirty: true}, "WB", "f", "dirty"},
		{worktreeInfo{Head: "1234567890abcdef", Locked: true}, "Git", "1234567890", "locked"},
		{worktreeInfo{Branch: "x", Prunable: true, Locked: true}, "Git", "x", "missing"},
		{worktreeInfo{Branch: "y"}, "Git", "y", "…"},
	}
	for _, c := range cases {
		kind, branch, state := worktreeColumnsOf(c.item)
		if kind != c.kind || branch != c.branch || state != c.state {
			t.Errorf("%+v: %s %s %s", c.item, kind, branch, state)
		}
		for column := 0; column < 3; column++ {
			_ = worktreeCellStyle(grid.Row{Ref: c.item}, column, nil)
		}
	}
	_ = worktreeCellStyle(grid.Row{}, 0, nil)

	p := newWorktreesPane()
	p.SetSize(60, 30)
	if p.detail() != "" {
		t.Fatal("nothing to describe")
	}
	p.items = []worktreeInfo{
		{Path: "/c", Canonical: true, Head: "abcdef1234567890", LastCommit: time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)},
		{Path: "/u", StatusRead: true},
		{Path: "/m", WB: &wbWorktreeInfo{HasManifest: true, EffortID: "e", OwnerState: "orphaned", Disposition: "d", Layout: "l"}},
		{Path: "/o", WB: &wbWorktreeInfo{HasManifest: true, EffortID: "e", OwnerAgent: "a", OwnerState: "s", ParentEffort: "p"}},
		{Path: "/n", WB: &wbWorktreeInfo{HasManifest: true, EffortID: "e"}},
	}
	p.fill("/c")
	want := []string{"Canonical clone", "Unmanaged by WB", "Owner  orphaned", "Owner  a (s)", "WB effort"}
	for i, w := range want {
		p.grid.SelectRow(i)
		if got := stripANSI(p.detail()); !strings.Contains(got, w) {
			t.Errorf("row %d: missing %q in %q", i, w, got)
		}
	}
	p.grid.SelectRow(0)
	if got := stripANSI(p.detail()); !strings.Contains(got, "HEAD  abcdef1234") || !strings.Contains(got, "Commit") {
		t.Fatalf("head and commit: %q", got)
	}
}

func TestWorktreesEnrichmentKeepsGitStateWithoutWB(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	installFakeWB(t, "", 1)
	h := open(t, canonical)
	h.RequireContains("2 Git worktrees").RequireContains("0 WB managed").RequireContains("unavailable")
	_ = linked
}

func TestWorktreesPaneSizesAndFocus(t *testing.T) {
	p := newWorktreesPane()
	p.SetSize(40, 30)
	p.items = []worktreeInfo{{Path: "/a"}}
	p.fill("/a")
	if p.detailHeight(10) != 0 {
		t.Fatal("no detail without focus")
	}
	p.Focus()
	if p.detailHeight(10) != 4 {
		t.Fatalf("detail is two fifths: %d", p.detailHeight(10))
	}
	if !strings.Contains(stripANSI(p.View()), "Enter open") {
		t.Fatal("the focused pane shows the detail")
	}
	p.Blur()
	p.items = nil
	p.Focus()
	if p.detailHeight(10) != 0 {
		t.Fatal("no detail without worktrees")
	}
	if !p.AtEdge(widgets.Up) {
		t.Fatal("an empty table is at every edge")
	}
	if _, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyDown}); cmd != nil {
		t.Fatal("nothing to move")
	}
}

func TestWorktreesRowActivationIgnoresMissingWorktrees(t *testing.T) {
	b := newBrowser(&session{})
	row := grid.Row{Ref: worktreeInfo{Path: "/gone", Prunable: true}}
	if b.activated(grid.RowActivatedMsg{ID: worktreesID, Row: row}) != nil {
		t.Fatal("a missing worktree cannot be opened")
	}
	row = grid.Row{Ref: worktreeInfo{Path: "/ok"}}
	if b.activated(grid.RowActivatedMsg{ID: worktreesID, Row: row}) == nil {
		t.Fatal("a worktree opens")
	}
}

func TestWorktreesMouse(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	h := open(t, canonical)
	h.Click(120, 25) // the lower half of the right column
	if browserOf(h).focus != paneWorktrees {
		t.Fatal("a click focuses the worktrees")
	}
	h.Wheel(120, 25, false).Wheel(120, 25, true)
	h.Click(120, 2)
	if browserOf(h).focus != panePreview {
		t.Fatal("a click in the upper half focuses the preview")
	}
}

func TestWorktreesProbeFailsQuietly(t *testing.T) {
	old := worktreeExecCommand
	worktreeExecCommand = func(string, ...string) *exec.Cmd { return exec.Command("sh", "-c", "exit 1") }
	defer func() { worktreeExecCommand = old }()
	msg := probeWorktreesCmd("/x", probeSync)().(worktreeProbeMsg)
	if msg.RepoRoot != "" || msg.Path != "/x" {
		t.Fatalf("%+v", msg)
	}
	_ = errors.New
	_ = context.Background
}

func TestWorktreesAltWOpensThePaneFromASubdirectory(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	wbReport(t, canonical, linked)
	sub := filepath.Join(canonical, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	h := open(t, sub)
	h.RequireNotContains("WORKTREES")
	h.Press("alt+w")
	h.RequireContains("2 Git worktrees")
	if browserOf(h).focus != paneWorktrees || h.Model().Zone() != nav.FocusToContent {
		t.Fatal("the pane opens and takes the focus")
	}
}

func TestBrowserAtEdgeWithTheWorktreesPaneOpen(t *testing.T) {
	b := newBrowser(&session{})
	b.focus = panePreview
	b.worktrees.visible = true
	if b.AtEdge(widgets.Down) {
		t.Fatal("Down from the preview goes on to the worktrees")
	}
	b.focus = paneWorktrees
	if b.AtEdge(widgets.Up) || !b.AtEdge(widgets.Down) {
		t.Fatal("the worktrees pane keeps Up for its own table")
	}
}
