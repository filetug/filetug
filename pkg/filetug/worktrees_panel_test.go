package filetug

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func runWorktreeTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newWorktreeTestRepository(t *testing.T) (canonical, linked string) {
	t.Helper()
	root := t.TempDir()
	canonical = filepath.Join(root, "repo")
	linked = filepath.Join(root, "linked")
	if err := os.Mkdir(canonical, 0o755); err != nil {
		t.Fatal(err)
	}
	runWorktreeTestGit(t, canonical, "init", "-b", "main")
	runWorktreeTestGit(t, canonical, "config", "user.email", "filetug@example.test")
	runWorktreeTestGit(t, canonical, "config", "user.name", "FileTug Test")
	if err := os.WriteFile(filepath.Join(canonical, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runWorktreeTestGit(t, canonical, "add", "README.md")
	runWorktreeTestGit(t, canonical, "commit", "-m", "initial")
	runWorktreeTestGit(t, canonical, "worktree", "add", "-b", "feature", linked)
	canonical, _ = filepath.EvalSymlinks(canonical)
	linked, _ = filepath.EvalSymlinks(linked)
	return canonical, linked
}

func installFakeWB(t *testing.T, output string, exitCode int) {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' '" + strings.ReplaceAll(output, "'", "'\\''") + "'\n"
	if exitCode != 0 {
		script += "exit " + string(rune('0'+exitCode)) + "\n"
	}
	path := filepath.Join(binDir, "wb")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitForPanelUpdates(t *testing.T, updates <-chan struct{}, count int) {
	t.Helper()
	for range count {
		select {
		case <-updates:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for worktrees panel update")
		}
	}
}

func TestParseGitWorktrees(t *testing.T) {
	canonical := filepath.Clean("/tmp/canonical")
	linked := filepath.Clean("/tmp/linked")
	output := "worktree " + linked + "\nHEAD 1234567890abcdef\ndetached\nlocked reason\nprunable reason\n\n" +
		"worktree " + canonical + "\nHEAD abcdef1234567890\nbranch refs/heads/main\n\n"
	items, err := parseGitWorktrees(output, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !items[0].Canonical || items[0].Branch != "main" {
		t.Fatalf("unexpected parsed worktrees: %+v", items)
	}
	if !items[1].Detached || !items[1].Locked || !items[1].Prunable {
		t.Fatalf("linked state not parsed: %+v", items[1])
	}
	if _, err := parseGitWorktrees("", canonical); err == nil {
		t.Fatal("expected empty worktree report to fail")
	}
}

func TestGitWorktreeInventory(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	resolved, items, err := readGitWorktrees(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != canonical || len(items) != 2 || items[1].Path != linked {
		t.Fatalf("unexpected inventory: canonical=%s items=%+v", resolved, items)
	}

	enrichGitWorktrees(context.Background(), items)
	for _, item := range items {
		if !item.StatusRead || item.LastCommit.IsZero() {
			t.Fatalf("worktree was not enriched: %+v", item)
		}
	}
	items = append(items, worktreeInfo{Path: filepath.Join(t.TempDir(), "missing")}, worktreeInfo{Prunable: true})
	enrichGitWorktrees(context.Background(), items)

	if got := gitRepositoryRoot(linked); got != linked {
		t.Fatalf("linked root = %q", got)
	}
	if got := gitRepositoryRoot(t.TempDir()); got != "" {
		t.Fatalf("non-repository root = %q", got)
	}
	if _, err := canonicalWorktreePath(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected a non-repository to fail canonical resolution")
	}

	bare := filepath.Join(t.TempDir(), "bare.git")
	if out, err := exec.Command("git", "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}
	if _, err := canonicalWorktreePath(context.Background(), bare); err == nil {
		t.Fatal("expected bare repository common directory to be unsupported")
	}
}

func TestWBWorktreeMetadata(t *testing.T) {
	canonical := filepath.Join(t.TempDir(), "org", "repo")
	managedPath := filepath.Join(t.TempDir(), "managed")
	report := wbOrphanReport{Families: []wbWorktreeFamily{{Worktrees: []wbWorktreeInfo{{
		Path: managedPath, CanonicalDir: canonical, HasManifest: true, EffortID: "effort",
	}}}}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	installFakeWB(t, string(encoded), 0)
	metadata, status := readWBWorktreeMetadata(context.Background(), canonical)
	if status != "WB metadata loaded" || !metadata[filepath.Clean(managedPath)].HasManifest {
		t.Fatalf("metadata=%+v status=%q", metadata, status)
	}

	otherCanonical := canonical + "-other"
	metadata, _ = readWBWorktreeMetadata(context.Background(), otherCanonical)
	if len(metadata) != 0 {
		t.Fatalf("unexpected metadata for another canonical: %+v", metadata)
	}
}

func TestWBWorktreeMetadataFallbacks(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		metadata, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if len(metadata) != 0 || status != "WB not installed; showing raw Git" {
			t.Fatalf("metadata=%+v status=%q", metadata, status)
		}
	})
	t.Run("command fails", func(t *testing.T) {
		installFakeWB(t, "", 1)
		_, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if status != "WB metadata unavailable" {
			t.Fatalf("status=%q", status)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		installFakeWB(t, "not-json", 0)
		_, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if status != "WB returned unsupported JSON" {
			t.Fatalf("status=%q", status)
		}
	})
}

func TestWorktreesPanelLoadAndPresentation(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
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

	updates := make(chan struct{}, 8)
	app := &recordApp{queueUpdateDraw: func(f func()) {
		f()
		updates <- struct{}{}
	}}
	nav := NewNavigator(app, withSkipAsyncFavoritesLoad())
	p := nav.worktrees
	p.Load(canonical)
	waitForPanelUpdates(t, updates, 2)
	if p.loading || len(p.items) != 2 || p.selected {
		t.Fatalf("unexpected loaded state: loading=%v selected=%v items=%+v", p.loading, p.selected, p.items)
	}
	if p.items[1].WB == nil || p.items[1].WB.EffortID != "worktrees" {
		t.Fatalf("WB enrichment missing: %+v", p.items)
	}

	p.ensureLoaded(canonical)
	p.table.Focus(func(tview.Primitive) {})
	if !p.focused || !p.selected {
		t.Fatalf("panel did not activate: focused=%v selected=%v", p.focused, p.selected)
	}
	p.table.Blur()
	if p.focused {
		t.Fatal("panel did not blur")
	}

	p.items = append(p.items,
		worktreeInfo{Path: "/tmp/dirty", Branch: "dirty", Dirty: true, StatusRead: true, Locked: true},
		worktreeInfo{Path: "/tmp/missing", Head: "1234567890abcdef", Prunable: true},
		worktreeInfo{Path: "/tmp/owner", Branch: "owner", WB: &wbWorktreeInfo{HasManifest: true, EffortID: "owner", OwnerState: "orphaned"}},
	)
	p.renderTable(canonical)
	for index := range p.items {
		p.selected = true
		p.renderDetail(index)
	}
	p.renderDetail(-1)
	p.selected = false
	p.renderDetail(0)
	p.clearDetail()
	var nilPanel *worktreesPanel
	nilPanel.clearDetail()

	input := p.table.InputHandler()
	opened := ""
	p.openPath = func(path string) { opened = path }
	p.table.SetSelectable(true, false)
	p.table.Select(1, 0)
	p.selected = false
	input(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	p.selected = true
	p.items[0].Prunable = true
	input(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	p.items[0].Prunable = false
	input(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	if opened != p.items[0].Path {
		t.Fatalf("opened path = %q", opened)
	}
	input(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone), func(tview.Primitive) {})
	waitForPanelUpdates(t, updates, 2)
	p.selected = true
	p.table.Select(1, 0)
	input(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), func(tview.Primitive) {})
	input(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), func(tview.Primitive) {})
	input(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone), func(tview.Primitive) {})

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.loading = true
	p.stopLoading()
	p.stopLoading()
	if ctx.Err() == nil {
		t.Fatal("stopLoading did not cancel")
	}
}

func TestWorktreesPanelNavigation(t *testing.T) {
	canonical, _ := newWorktreeTestRepository(t)
	app := &recordApp{}
	nav := NewNavigator(app, withSkipAsyncFavoritesLoad())

	var nilNav *Navigator
	nilNav.showWorktreesPanel()
	nilNav.showWorktreesPanelAtRepositoryRoot(canonical)

	nav.current.SetDir(nil)
	nav.showWorktreesPanel()
	nonRepo := t.TempDir()
	nav.current.SetDir(files.NewDirContext(nav.store, nonRepo, nil))
	nav.showWorktreesPanel()
	if !nav.worktrees.visible {
		t.Fatal("expected not-in-repository panel")
	}

	nav.worktrees.repoRoot = canonical
	nav.worktrees.items = []worktreeInfo{{Path: canonical, Canonical: true, Branch: "main"}}
	nav.right.content = nav.worktrees
	nav.showWorktreesPanel()
	if len(app.focusCalls) == 0 {
		t.Fatal("visible panel was not focused")
	}

	nav.right.content = nav.previewer
	nav.current.SetDir(files.NewDirContext(nav.store, canonical, nil))
	nav.showWorktreesPanel()
	if nav.right.content != nav.worktrees {
		t.Fatal("repository panel was not shown")
	}

	nav.worktrees.visible = true
	nav.right.content = nav.worktrees
	nav.right.SetContent(nav.previewer)
	if nav.right.content != nav.worktrees {
		t.Fatal("visible worktrees panel was overwritten")
	}

	if got := nav.inputCapture(tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModAlt)); got != nil {
		t.Fatalf("Option-W was not consumed: %v", got)
	}

	nav.showWorktreesPanelAtRepositoryRoot(nonRepo)
	nav.showWorktreesPanelAtRepositoryRoot(filepath.Dir(canonical))
	nav.showWorktreesPanelAtRepositoryRoot(canonical)

	nav.store = &stubStore{root: url.URL{Scheme: "http"}}
	nav.showWorktreesPanel()
	nav.showWorktreesPanelAtRepositoryRoot(canonical)
}

func TestWorktreePresentationHelpers(t *testing.T) {
	items := []worktreeInfo{{Branch: "z"}, {Branch: "a"}, {Branch: "main", Canonical: true}}
	sortWorktrees(items)
	if !items[0].Canonical || items[1].Branch != "a" {
		t.Fatalf("unexpected sort: %+v", items)
	}
	if got := repositoryLabel("/tmp/org/repo"); got != "org/repo" {
		t.Fatalf("label=%q", got)
	}
	if got := shortSHA("1234567890abcdef"); got != "1234567890" {
		t.Fatalf("short SHA=%q", got)
	}
	if got := shortSHA("short"); got != "short" {
		t.Fatalf("short SHA=%q", got)
	}
}

func TestWorktreesPanelLoadError(t *testing.T) {
	updates := make(chan struct{}, 2)
	app := &recordApp{queueUpdateDraw: func(f func()) {
		f()
		updates <- struct{}{}
	}}
	nav := NewNavigator(app, withSkipAsyncFavoritesLoad())
	nav.worktrees.ensureLoaded(t.TempDir())
	waitForPanelUpdates(t, updates, 1)
	if nav.worktrees.loading {
		t.Fatal("failed load remained active")
	}

	var nilPanel *worktreesPanel
	nilPanel.Load("/tmp")
	empty := &worktreesPanel{}
	empty.Load("/tmp")
	empty.selectDefaultRow()
	var nilSelectable *worktreesPanel
	nilSelectable.selectDefaultRow()
}

func TestReadGitWorktreesFailures(t *testing.T) {
	withTestGlobalLock(t)
	oldCommandContext := worktreeExecCommandContext
	t.Cleanup(func() { worktreeExecCommandContext = oldCommandContext })

	t.Run("canonical", func(t *testing.T) {
		worktreeExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sh", "-c", "exit 1")
		}
		if _, _, err := readGitWorktrees(context.Background(), "/tmp/repo"); err == nil {
			t.Fatal("expected canonical resolution to fail")
		}
	})

	for _, test := range []struct {
		name   string
		script string
	}{
		{name: "list", script: "exit 1"},
		{name: "empty", script: "printf ''"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			worktreeExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				calls++
				if calls == 1 {
					return exec.CommandContext(ctx, "sh", "-c", "printf '/tmp/repo/.git'")
				}
				return exec.CommandContext(ctx, "sh", "-c", test.script)
			}
			if _, _, err := readGitWorktrees(context.Background(), "/tmp/repo"); err == nil {
				t.Fatal("expected worktree inventory to fail")
			}
		})
	}
}

func TestWorktreesPanelIgnoresStaleUpdates(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	report := wbOrphanReport{Families: []wbWorktreeFamily{{Worktrees: []wbWorktreeInfo{{
		Path: linked, CanonicalDir: canonical, HasManifest: true, LastCommit: time.Now().Add(-time.Hour),
	}}}}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	installFakeWB(t, string(encoded), 0)

	callbacks := make(chan func(), 4)
	app := &recordApp{queueUpdateDraw: func(f func()) { callbacks <- f }}
	nav := NewNavigator(app, withSkipAsyncFavoritesLoad())
	p := nav.worktrees
	p.Load(t.TempDir())
	first := <-callbacks
	p.loadID.Add(1)
	first()
	p.stopLoading()

	p.Load(canonical)
	first = <-callbacks
	first()
	final := <-callbacks
	p.loadID.Add(1)
	final()
	p.stopLoading()
}
