package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/go-git/go-git/v5"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/uitest"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func testSummary(git *gitDirStatus) *dirSummary {
	return &dirSummary{Path: "/d", Git: git, Groups: summarize([]os.DirEntry{
		sized("a.go", 1500), sized("b.go", 2), sized("c.png", 3*1024*1024), sized("x.txt", 5*1024*1024*1024), sized("y.log", 2*1024*1024*1024*1024),
	})}
}

func TestSummaryRows(t *testing.T) {
	rows := typeRows(testSummary(nil).Groups)
	if len(rows) == 0 {
		t.Fatal("rows")
	}
	group := rows[0]
	if group.Values[0] != "▼ Code" || group.Values[1] != "" || !group.Ref.(typeRef).Group {
		t.Fatalf("a group with one extension shows no totals: %+v", group)
	}
	ext := rows[1]
	if ext.Values[0] != "  *.go" || ext.Values[1] != "2 files" || ext.Ref.(typeRef).Extensions[0] != ".go" {
		t.Fatalf("an extension: %+v", ext)
	}
	multi := typeRows(summarize([]os.DirEntry{sized("a.go", 1), sized("b.js", 2)}))
	if multi[0].Values[1] != "2 files" || multi[0].Values[2] != "3B" {
		t.Fatalf("a group with several extensions shows totals: %+v", multi[0])
	}
	none := typeRows(summarize([]os.DirEntry{sized("Makefile", 1)}))
	if none[1].Values[0] != "  <no extension>" || none[1].Values[1] != "1 file" {
		t.Fatalf("files without an extension: %+v", none[1])
	}
}

func TestSizeColors(t *testing.T) {
	for _, text := range []string{"1TB", "1GB", "1MB", "1KB", "12B"} {
		_ = sizeColor(text)
	}
	if typeCellStyle(grid.Row{}, 0, nil).GetBold() {
		t.Fatal("rows without a reference are plain")
	}
	if !typeCellStyle(grid.Row{Ref: typeRef{Group: true}}, 0, nil).GetBold() {
		t.Fatal("groups are bold")
	}
	_ = typeCellStyle(grid.Row{Ref: typeRef{}}, 2, "3MB")
	_ = typeCellStyle(grid.Row{Ref: typeRef{}}, 2, "")
	_ = typeCellStyle(grid.Row{Ref: typeRef{}}, 2, 5)
}

func TestChangeCellStyle(t *testing.T) {
	for _, badge := range []string{"A", "D", "M", "?"} {
		_ = changeCellStyle(grid.Row{Ref: gitEntry{Badge: gitBadge{Text: badge}}}, 1, nil)
	}
	_ = changeCellStyle(grid.Row{}, 1, nil)
	_ = changeCellStyle(grid.Row{Ref: gitEntry{}}, 0, nil)
}

func TestSummaryPaneTabsAndFilters(t *testing.T) {
	status := &gitDirStatus{RepoRoot: "/d", Entries: []gitEntry{{FullPath: "/d/a.go", DisplayName: "a.go", Badge: gitBadge{Text: "M", Label: "changed"}}}}
	p := newSummaryPane(testSummary(status))
	p.SetSize(50, 12)
	p.SetFocused(true)
	if p.active != tabGit {
		t.Fatal("a repository with changes opens on the git tab")
	}
	view := uitest.Plain(p.View())
	if !strings.Contains(view, "File types") || !strings.Contains(view, "a.go") || !strings.Contains(view, "M:changed") {
		t.Fatalf("git tab: %q", view)
	}
	if p.Filter() != nil {
		t.Fatal("the git tab does not filter")
	}
	if cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyLeft}); cmd == nil || p.active != tabTypes {
		t.Fatal("Left goes to the file types and filters again")
	}
	if group := msgOf(t, p.Selection()).(extFilterMsg); len(group.Extensions) != 1 || group.Extensions[0] != ".go" {
		t.Fatalf("the first row is the Code group: %+v", group)
	}
	if cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyLeft}); msgOf(t, cmd) != (focusFilesMsg{}) {
		t.Fatal("Left on the first tab returns to the file list")
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if p.active != tabGit {
		t.Fatal("Right goes to the git tab")
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if p.Update(tea.MouseClickMsg{}) != nil {
		t.Fatal("only keys are handled")
	}
	p.activate(tabTypes)
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	filter := msgOf(t, p.Filter()).(extFilterMsg)
	if len(filter.Extensions) != 1 || filter.Extensions[0] != ".go" {
		t.Fatalf("an extension row filters by it: %+v", filter)
	}
	if !p.AtEdge(widgets.Up) && p.AtEdge(widgets.Left) {
		t.Fatal("edges")
	}
}

func msgOf(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	return cmd()
}

func TestSummaryPaneWithoutGit(t *testing.T) {
	p := newSummaryPane(testSummary(nil))
	p.SetSize(40, 10)
	if p.active != tabTypes || p.Update(tea.KeyPressMsg{Code: tea.KeyRight}) != nil || p.active != tabTypes {
		t.Fatal("there is no git tab outside a repository")
	}
	if p.toggleStage() != nil {
		t.Fatal("nothing to stage on the file types tab")
	}
	p.activate(tabGit)
	if !strings.Contains(uitest.Plain(p.View()), "Not a git repository") {
		t.Fatal("the reason there is no status")
	}
	empty := &summaryPane{data: &dirSummary{Git: &gitDirStatus{}}, tabs: p.tabs, types: p.types, changes: p.changes, active: tabGit, w: 30, h: 5}
	if !strings.Contains(uitest.Plain(empty.View()), "No changes") {
		t.Fatal("a clean repository")
	}
	failed := &summaryPane{data: &dirSummary{Git: &gitDirStatus{Err: errors.New("broken")}}, tabs: p.tabs, types: p.types, changes: p.changes, active: tabGit, w: 30, h: 5}
	if !strings.Contains(uitest.Plain(failed.View()), "broken") {
		t.Fatal("a failure")
	}
	p.types.SetData(p.types.Columns(), nil)
	p.activate(tabTypes)
	if ext := msgOf(t, p.Filter()).(extFilterMsg); len(ext.Extensions) != 0 {
		t.Fatal("no row, no filter")
	}
}

func TestSummaryPaneStagesWithSpace(t *testing.T) {
	fakeRepoStatus(t, "/d", git.Status{"a.go": {Worktree: git.Modified}})
	stageInGit = func(string) error { return nil }
	status := &gitDirStatus{RepoRoot: "/d", Entries: []gitEntry{{FullPath: "/d/a.go", DisplayName: "a.go", Badge: gitBadge{Text: "M"}}}}
	p := newSummaryPane(testSummary(status))
	p.SetSize(40, 10)
	msg, ok := msgOf(t, p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})).(gitDirLoadedMsg)
	if !ok || msg.Path != "/d" {
		t.Fatalf("Space stages and reads the status again: %+v", msg)
	}
	p.GitLoaded(gitDirLoadedMsg{Path: "/other", Status: nil})
	if p.data.Git == nil {
		t.Fatal("a status for another directory is ignored")
	}
	p.GitLoaded(msg)
	if len(p.data.Git.Entries) != 1 {
		t.Fatal("a fresh status replaces the old one")
	}
	p.changes.SetData(p.changes.Columns(), nil)
	if p.toggleStage() != nil {
		t.Fatal("no row to stage")
	}
	p.changes.SetData(p.changes.Columns(), []grid.Row{{Key: "x"}})
	if p.toggleStage() != nil {
		t.Fatal("a row without an entry")
	}
}

func TestDirectorySummaryInTheBrowser(t *testing.T) {
	dir := tree(t)
	write(t, filepath.Join(dir, "alpha", "z.go"), "package z")
	write(t, filepath.Join(dir, "alpha", "y.md"), "# y")
	h := open(t, dir)
	h.Press("down", "down") // alpha
	h.RequireContains("File types").RequireContains("*.go").RequireContains("*.md")
	h.Press("right", "right") // the summary has the focus
	if browserOf(h).focus != panePreview {
		t.Fatal("focus in the summary")
	}
	h.Press("down") // the first extension: only files with it show in the list
	h.RequireContains("z.go").RequireNotContains("y.md ")
	h.Press("left") // back to the file list
	if browserOf(h).focus != paneFiles {
		t.Fatal("Left returns to the file list")
	}
	h.RequireContains("z.go")
}

func TestSummaryInBrowserIgnoresStrayMessages(t *testing.T) {
	b := newBrowser(&session{})
	if _, cmd := b.Update(grid.SelectionChangedMsg{ID: typesID}); cmd != nil {
		t.Fatal("no summary, no filter")
	}
	if _, cmd := b.Update(gitDirLoadedMsg{}); cmd != nil {
		t.Fatal("no summary, no status")
	}
	s, _ := b.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	b = s.(browser)
	b.preview.Show(previewMsg{Title: "d", Summary: testSummary(nil)})
	s, _ = b.Update(gitDirLoadedMsg{Path: "/d", Status: &gitDirStatus{}})
	if s.(browser).preview.summary.data.Git == nil {
		t.Fatal("the summary takes the fresh status")
	}
	if _, cmd := b.Update(grid.SelectionChangedMsg{ID: typesID}); cmd == nil {
		t.Fatal("the summary's cursor filters the file list")
	}
}
