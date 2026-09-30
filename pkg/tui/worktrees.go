package tui

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// worktreesID names the worktrees grid in its messages.
const worktreesID = "worktrees"

// worktreesTimeout bounds one inventory of a repository's worktrees.
const worktreesTimeout = time.Minute

// probeIntent says why the repository of a directory is being looked up.
type probeIntent int

const (
	// probeShow is Alt+W: show the pane and focus it.
	probeShow probeIntent = iota
	// probeSync keeps an open pane in step with the directory.
	probeSync
	// probeAuto opens the pane when the directory is a repository root.
	probeAuto
)

// worktreesToggleMsg is sent by Alt+W.
type worktreesToggleMsg struct{}

// worktreeProbeMsg reports the repository root of a directory, "" when the
// directory is not inside a repository.
type worktreeProbeMsg struct {
	Path     string
	RepoRoot string
	Intent   probeIntent
}

// probeWorktreesCmd looks up the repository of a directory off the event loop.
func probeWorktreesCmd(path string, intent probeIntent) tea.Cmd {
	return func() tea.Msg {
		return worktreeProbeMsg{Path: path, RepoRoot: gitRepositoryRoot(path), Intent: intent}
	}
}

// worktreesListedMsg is the registry of a repository's worktrees.
type worktreesListedMsg struct {
	ID        uint64
	Canonical string
	Items     []worktreeInfo
	Err       error
}

// worktreesEnrichedMsg adds the state of every worktree and the WB metadata.
type worktreesEnrichedMsg struct {
	ID       uint64
	Items    []worktreeInfo
	WBStatus string
}

// worktreesPane lists the Git worktrees of the repository of the current
// directory, with WB metadata when the wb tool is installed.
type worktreesPane struct {
	visible   bool
	repoRoot  string
	canonical string
	items     []worktreeInfo
	header    string
	status    string
	loading   bool
	loadID    uint64
	ctx       context.Context
	cancel    context.CancelFunc
	grid      *grid.Model
	focused   bool
	w, h      int
}

var worktreeColumns = []grid.Column{{Name: "Kind"}, {Name: "Branch", MaxWidth: 40}, {Name: "State"}}

func newWorktreesPane() worktreesPane {
	p := worktreesPane{header: worktreesHeader("")}
	p.grid = grid.New(worktreeColumns, nil,
		grid.WithID(worktreesID),
		grid.WithRowSelection(),
		grid.WithoutFrame(),
		grid.WithoutSwitcher(),
		grid.WithFilterDisabled(),
		grid.WithStyle(grid.StyleMinimal),
		grid.WithCellStyle(worktreeCellStyle),
		grid.WithFooterHook(func(*grid.Model, string) string { return "" }),
	)
	return p
}

// headerRows is the height of the title line and the two-line status.
const headerRows = 3

// worktreesHeader is the title line of the pane.
func worktreesHeader(repo string) string {
	title := lipgloss.NewStyle().Foreground(theme.FocusColor()).Bold(true).Render("WORKTREES")
	if repo == "" {
		return title
	}
	return title + " · " + lipgloss.NewStyle().Foreground(theme.MutedColor()).Render(repo)
}

// Text styles of the status line.
func warnText(s string) string {
	return lipgloss.NewStyle().Foreground(theme.AccentColor()).Render(s)
}

func badText(s string) string {
	return lipgloss.NewStyle().Foreground(theme.ErrorColor()).Render(s)
}

func goodText(s string) string {
	return lipgloss.NewStyle().Foreground(adaptive("#3BD17F", "#1B7F46")).Render(s)
}

// SetSize gives the pane its outer size; the frame is drawn by the caller.
func (p *worktreesPane) SetSize(w, h int) {
	p.w, p.h = w, h
	p.layout()
}

// layout sizes the table: below the header and status, and above the detail
// while the pane has focus.
func (p *worktreesPane) layout() {
	rest := max(p.h-headerRows, 0)
	p.grid.SetSize(p.w, rest-p.detailHeight(rest))
}

// detailHeight is how many rows the detail of the selected worktree takes.
func (p worktreesPane) detailHeight(rest int) int {
	if !p.focused || len(p.items) == 0 {
		return 0
	}
	return rest * 2 / 5
}

// Focus and Blur tell the pane whether it has the focus.
func (p *worktreesPane) Focus() { p.focused = true; p.layout() }
func (p *worktreesPane) Blur()  { p.focused = false; p.layout() }

// Hide closes the pane and stops what it is reading.
func (p *worktreesPane) Hide() {
	p.visible = false
	p.stop()
}

// stop cancels the read in flight.
func (p *worktreesPane) stop() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.loading = false
}

// ShowNoRepository shows that the directory is not in a repository.
func (p *worktreesPane) ShowNoRepository() {
	p.stop()
	p.visible = true
	p.items = nil
	p.header = worktreesHeader("")
	p.status = warnText("Not inside a Git repository")
	p.grid.SetData(worktreeColumns, nil)
	p.layout()
}

// EnsureLoaded reads the worktrees of a repository unless they are loaded or
// being loaded already.
func (p *worktreesPane) EnsureLoaded(repoRoot string) tea.Cmd {
	clean := filepath.Clean(repoRoot)
	if p.repoRoot == clean && (p.loading || len(p.items) > 0) {
		return nil
	}
	return p.Load(clean)
}

// Load reads the worktrees of a repository.
func (p *worktreesPane) Load(repoRoot string) tea.Cmd {
	p.stop()
	ctx, cancel := context.WithTimeout(context.Background(), worktreesTimeout)
	p.ctx, p.cancel = ctx, cancel
	p.loading = true
	p.loadID++
	id := p.loadID
	p.repoRoot = filepath.Clean(repoRoot)
	p.items = nil
	p.header = worktreesHeader(repositoryLabel(repoRoot))
	p.status = warnText("Loading Git worktree registry…")
	p.grid.SetData(worktreeColumns, nil)
	p.layout()
	return func() tea.Msg {
		canonical, items, err := readGitWorktrees(ctx, repoRoot)
		return worktreesListedMsg{ID: id, Canonical: canonical, Items: items, Err: err}
	}
}

// Listed shows the registry of worktrees and starts reading their state.
func (p *worktreesPane) Listed(msg worktreesListedMsg) tea.Cmd {
	if msg.ID != p.loadID {
		return nil
	}
	if msg.Err != nil {
		p.stop()
		p.status = badText("Git worktrees unavailable: " + msg.Err.Error())
		return nil
	}
	p.canonical = msg.Canonical
	p.items = msg.Items
	p.status = goodText(fmt.Sprintf("%d Git worktrees", len(msg.Items))) + " · loading status and WB metadata…"
	p.fill(p.repoRoot)
	ctx, id, canonical := p.ctx, p.loadID, msg.Canonical
	items := slices.Clone(msg.Items)
	return func() tea.Msg {
		wbByPath, wbStatus := enrichWorktrees(ctx, canonical, items)
		for index := range items {
			if wb, ok := wbByPath[filepath.Clean(items[index].Path)]; ok {
				items[index].WB = &wb
				if !wb.LastCommit.IsZero() {
					items[index].LastCommit = wb.LastCommit
				}
				items[index].Dirty = wb.Dirty
				items[index].StatusRead = !wb.Missing
			}
		}
		return worktreesEnrichedMsg{ID: id, Items: items, WBStatus: wbStatus}
	}
}

// enrichWorktrees reads the state of every worktree and the WB metadata in
// parallel.
func enrichWorktrees(ctx context.Context, canonical string, items []worktreeInfo) (map[string]wbWorktreeInfo, string) {
	var wg sync.WaitGroup
	var wbByPath map[string]wbWorktreeInfo
	var wbStatus string
	wg.Go(func() { enrichGitWorktrees(ctx, items) })
	wg.Go(func() { wbByPath, wbStatus = readWBWorktreeMetadata(ctx, canonical) })
	wg.Wait()
	return wbByPath, wbStatus
}

// Enriched shows the state of every worktree.
func (p *worktreesPane) Enriched(msg worktreesEnrichedMsg) {
	if msg.ID != p.loadID {
		return
	}
	p.stop()
	p.items = msg.Items
	managed := 0
	for _, item := range msg.Items {
		if item.WB != nil && item.WB.HasManifest {
			managed++
		}
	}
	p.status = goodText(fmt.Sprintf("%d Git worktrees", len(msg.Items))) +
		" · " + lipgloss.NewStyle().Foreground(theme.FocusColor()).Render(fmt.Sprintf("%d WB managed", managed)) +
		" · " + msg.WBStatus
	p.fill(p.repoRoot)
	p.layout()
}

// fill puts the worktrees in the table and moves the cursor to the worktree
// the user is in.
func (p *worktreesPane) fill(current string) {
	rows := make([]grid.Row, len(p.items))
	selected := 0
	for i, item := range p.items {
		kind, branch, state := worktreeColumnsOf(item)
		rows[i] = grid.Row{Key: item.Path, Values: []any{kind, branch, state}, Ref: item}
		if filepath.Clean(item.Path) == filepath.Clean(current) || filepath.Clean(item.Path) == filepath.Clean(p.repoRoot) {
			selected = i
		}
	}
	p.grid.SetData(worktreeColumns, rows)
	if len(rows) > 0 {
		p.grid.SelectRow(selected)
	}
	p.layout()
}

// worktreeColumnsOf is the kind, branch and state of a worktree.
func worktreeColumnsOf(item worktreeInfo) (kind, branch, state string) {
	kind = "Git"
	switch {
	case item.Canonical:
		kind = "Clone"
	case item.WB != nil && item.WB.HasManifest:
		kind = "WB"
	}
	branch = item.Branch
	if branch == "" {
		branch = shortSHA(item.Head)
	}
	state = "…"
	if item.StatusRead {
		state = "clean"
		if item.Dirty {
			state = "dirty"
		}
	}
	if item.Locked {
		state = "locked"
	}
	if item.Prunable {
		state = "missing"
	}
	return kind, branch, state
}

// worktreeCellStyle colours the kind and the state of a worktree.
func worktreeCellStyle(row grid.Row, column int, _ any) lipgloss.Style {
	item, ok := row.Ref.(worktreeInfo)
	if !ok {
		return lipgloss.NewStyle()
	}
	kind, _, state := worktreeColumnsOf(item)
	switch column {
	case 0:
		return lipgloss.NewStyle().Foreground(kindColor(kind))
	case 2:
		return lipgloss.NewStyle().Foreground(stateColor(state))
	}
	return lipgloss.NewStyle()
}

func kindColor(kind string) color.Color {
	switch kind {
	case "Clone":
		return theme.FocusColor()
	case "WB":
		return adaptive("#B48EFF", "#6B3FB8")
	}
	return theme.MutedColor()
}

func stateColor(state string) color.Color {
	switch state {
	case "clean":
		return adaptive("#3BD17F", "#1B7F46")
	case "dirty":
		return adaptive("#FFA500", "#9A5B00")
	case "locked":
		return theme.AccentColor()
	case "missing":
		return theme.ErrorColor()
	}
	return theme.MutedColor()
}

// Current is the worktree under the cursor.
func (p worktreesPane) Current() (worktreeInfo, bool) {
	row, ok := p.grid.CurrentRow()
	if !ok {
		return worktreeInfo{}, false
	}
	item, ok := row.Ref.(worktreeInfo)
	return item, ok
}

// Update forwards a message to the table.
func (p worktreesPane) Update(msg tea.Msg) (worktreesPane, tea.Cmd) {
	var cmd tea.Cmd
	p.grid, cmd = p.grid.Update(msg)
	return p, cmd
}

// AtEdge implements widgets.Boundary.
func (p worktreesPane) AtEdge(dir widgets.Direction) bool { return p.grid.AtEdge(dir) }

// detail describes the worktree under the cursor.
func (p worktreesPane) detail() string {
	item, ok := p.Current()
	if !ok {
		return ""
	}
	muted := lipgloss.NewStyle().Foreground(theme.MutedColor())
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(item.Path) + "\n")
	if item.Canonical {
		sb.WriteString(lipgloss.NewStyle().Foreground(theme.FocusColor()).Render("Canonical clone") + "\n")
	} else {
		sb.WriteString("Git linked worktree\n")
	}
	if item.Head != "" {
		fmt.Fprintf(&sb, "HEAD  %s\n", shortSHA(item.Head))
	}
	if !item.LastCommit.IsZero() {
		fmt.Fprintf(&sb, "Commit %s\n", item.LastCommit.Local().Format("2006-01-02 15:04"))
	}
	if item.WB == nil || !item.WB.HasManifest {
		if !item.Canonical {
			sb.WriteString("\n" + muted.Render("Unmanaged by WB"))
		}
	} else {
		wb := item.WB
		fmt.Fprintf(&sb, "\n%s %s\n", lipgloss.NewStyle().Foreground(theme.FocusColor()).Bold(true).Render("WB effort"), wb.EffortID)
		if wb.ParentEffort != "" {
			fmt.Fprintf(&sb, "Parent %s\n", wb.ParentEffort)
		}
		switch {
		case wb.OwnerAgent != "":
			fmt.Fprintf(&sb, "Owner  %s (%s)\n", wb.OwnerAgent, wb.OwnerState)
		case wb.OwnerState != "":
			fmt.Fprintf(&sb, "Owner  %s\n", wb.OwnerState)
		}
		fmt.Fprintf(&sb, "State  %s · %s\n", wb.Disposition, wb.Layout)
	}
	sb.WriteString("\n" + muted.Render("Enter open · r refresh · esc close"))
	return sb.String()
}

// View draws the header, the status, the table and, while focused, the detail.
func (p worktreesPane) View() string {
	rest := max(p.h-headerRows, 0)
	detailH := p.detailHeight(rest)
	table := widgets.Fit(p.grid.View(p.w, p.focused), p.w, rest-detailH)
	status := widgets.Fit(lipgloss.NewStyle().Width(p.w).Render(p.status), p.w, 2)
	parts := []string{widgets.Fit(p.header, p.w, 1), status, table}
	if detailH > 0 {
		wrapped := lipgloss.NewStyle().Width(p.w).Render(p.detail())
		parts = append(parts, widgets.Fit(wrapped, p.w, detailH))
	}
	return strings.Join(parts, "\n")
}
