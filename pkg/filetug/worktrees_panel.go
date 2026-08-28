package filetug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/themes"
)

type worktreeInfo struct {
	Path       string
	Head       string
	Branch     string
	Canonical  bool
	Detached   bool
	Locked     bool
	Prunable   bool
	Dirty      bool
	StatusRead bool
	LastCommit time.Time
	WB         *wbWorktreeInfo
}

type wbWorktreeInfo struct {
	Path         string    `json:"path"`
	CanonicalDir string    `json:"canonical_dir"`
	Repository   string    `json:"repository"`
	Layout       string    `json:"layout"`
	Branch       string    `json:"branch"`
	EffortID     string    `json:"effort_id"`
	ParentEffort string    `json:"parent_effort,omitempty"`
	RootEffort   string    `json:"root_effort"`
	HasManifest  bool      `json:"has_manifest"`
	Provenance   string    `json:"provenance,omitempty"`
	LastCommit   time.Time `json:"last_commit,omitempty"`
	Dirty        bool      `json:"dirty"`
	Missing      bool      `json:"missing"`
	Merged       bool      `json:"merged_into_target"`
	OwnerState   string    `json:"owner_state"`
	OwnerAgent   string    `json:"owner_agent,omitempty"`
	OwnerPID     int       `json:"owner_pid,omitempty"`
	Disposition  string    `json:"disposition"`
	Evidence     []string  `json:"evidence"`
}

type wbWorktreeFamily struct {
	Worktrees []wbWorktreeInfo `json:"worktrees"`
}

type wbOrphanReport struct {
	Families []wbWorktreeFamily `json:"families"`
}

type worktreesPanel struct {
	*sneatv.Boxed
	nav       *Navigator
	rows      *tview.Flex
	header    *tview.TextView
	status    *tview.TextView
	table     *tview.Table
	detail    *tview.TextView
	repoRoot  string
	canonical string
	items     []worktreeInfo
	visible   bool
	focused   bool
	selected  bool
	loading   bool
	cancel    context.CancelFunc
	loadID    atomic.Uint64
}

func newWorktreesPanel(nav *Navigator) *worktreesPanel {
	header := tview.NewTextView().SetDynamicColors(true)
	status := tview.NewTextView().SetDynamicColors(true)
	table := tview.NewTable().SetSelectable(false, false)
	table.SetSelectedStyle(themes.CurrentTheme.BlurredSelectedTextStyle())
	table.SetFixed(1, 0)
	detail := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	detail.SetBorderPadding(0, 0, 1, 1)

	rows := tview.NewFlex().SetDirection(tview.FlexRow)
	rows.AddItem(header, 1, 0, false)
	rows.AddItem(status, 1, 0, false)
	rows.AddItem(table, 0, 3, true)
	rows.AddItem(detail, 0, 0, false)

	p := &worktreesPanel{
		nav: nav, rows: rows, header: header, status: status, table: table, detail: detail,
		Boxed: sneatv.NewBoxed(rows, sneatv.WithLeftBorder(0, -1)),
	}
	p.SetTitle("Worktrees")
	p.header.SetText("[#7aa2f7::b] WORKTREES[-]")
	p.table.SetSelectionChangedFunc(func(row, _ int) {
		if p.selected {
			p.renderDetail(row - 1)
		}
	})
	p.table.SetSelectedFunc(func(row, _ int) {
		index := row - 1
		if !p.selected || index < 0 || index >= len(p.items) || p.items[index].Prunable {
			return
		}
		p.nav.goDirByPath(p.items[index].Path)
	})
	p.table.SetFocusFunc(func() {
		p.focused = true
		p.table.SetSelectable(true, false)
		p.table.SetSelectedStyle(themes.CurrentTheme.FocusedSelectedTextStyle())
		p.selectDefaultRow()
		nav.activeCol = 2
	})
	p.table.SetBlurFunc(func() {
		p.focused = false
		p.table.SetSelectedStyle(themes.CurrentTheme.BlurredSelectedTextStyle())
	})
	p.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			p.visible = false
			nav.right.SetContent(nav.previewer)
			nav.app.SetFocus(nav.files)
			return nil
		case tcell.KeyLeft:
			nav.app.SetFocus(nav.files)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'r' || event.Rune() == 'R' {
				p.Load(p.repoRoot)
				return nil
			}
		}
		return event
	})
	return p
}

func (nav *Navigator) showWorktreesPanel() {
	if nav == nil || nav.worktrees == nil || nav.store == nil || nav.store.RootURL().Scheme != "file" {
		return
	}
	if nav.right != nil && nav.right.content == nav.worktrees && nav.worktrees.repoRoot != "" {
		nav.worktrees.visible = true
		nav.right.SetContent(nav.worktrees)
		nav.app.SetFocus(nav.worktrees.table)
		return
	}
	path := nav.currentDirPath()
	if path == "" {
		return
	}
	repoRoot := gitRepositoryRoot(path)
	if repoRoot == "" {
		nav.worktrees.visible = true
		nav.worktrees.SetTitle("Worktrees")
		nav.worktrees.header.SetText("[#7aa2f7::b] WORKTREES[-]")
		nav.worktrees.status.SetText("[yellow]Not inside a Git repository[-]")
		nav.worktrees.table.Clear()
		nav.worktrees.clearDetail()
		nav.right.SetContent(nav.worktrees)
		return
	}
	nav.worktrees.visible = true
	nav.right.SetContent(nav.worktrees)
	nav.worktrees.ensureLoaded(repoRoot)
	nav.app.SetFocus(nav.worktrees.table)
}

func (nav *Navigator) showWorktreesPanelAtRepositoryRoot(path string) {
	if nav == nil || nav.worktrees == nil || nav.store == nil || nav.store.RootURL().Scheme != "file" {
		return
	}
	repoRoot := gitRepositoryRoot(path)
	if repoRoot == "" || filepath.Clean(repoRoot) != filepath.Clean(path) {
		return
	}
	nav.worktrees.visible = true
	nav.right.SetContent(nav.worktrees)
	nav.worktrees.ensureLoaded(repoRoot)
}

func (p *worktreesPanel) ensureLoaded(repoRoot string) {
	clean := filepath.Clean(repoRoot)
	if p.repoRoot == clean && (p.loading || len(p.items) > 0) {
		return
	}
	p.Load(clean)
}

func (p *worktreesPanel) Load(repoRoot string) {
	if p == nil || p.nav == nil || p.nav.app == nil {
		return
	}
	p.stopLoading()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	p.cancel = cancel
	p.loading = true
	loadID := p.loadID.Add(1)
	cleanRepoRoot := filepath.Clean(repoRoot)
	if p.repoRoot != cleanRepoRoot {
		p.selected = false
		p.table.SetSelectable(p.focused, false)
	}
	p.repoRoot = cleanRepoRoot
	p.SetTitle("Worktrees")
	p.header.SetText("[#7aa2f7::b] WORKTREES[-] · [gray]" + tview.Escape(repositoryLabel(repoRoot)) + "[-]")
	p.status.SetText("[yellow]Loading Git worktree registry…[-]")
	p.table.Clear()
	p.clearDetail()

	go func() {
		defer cancel()
		canonical, items, err := readGitWorktrees(ctx, repoRoot)
		initialItems := append([]worktreeInfo(nil), items...)
		p.nav.app.QueueUpdateDraw(func() {
			if loadID != p.loadID.Load() {
				return
			}
			if err != nil {
				p.loading = false
				p.status.SetText("[red]Git worktrees unavailable: " + tview.Escape(err.Error()) + "[-]")
				return
			}
			p.canonical = canonical
			p.items = initialItems
			p.status.SetText(fmt.Sprintf("[green]%d Git worktrees[-] · loading status and WB metadata…", len(items)))
			p.renderTable(repoRoot)
		})
		if err != nil {
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		var wbByPath map[string]wbWorktreeInfo
		var wbStatus string
		go func() {
			defer wg.Done()
			enrichGitWorktrees(ctx, items)
		}()
		go func() {
			defer wg.Done()
			wbByPath, wbStatus = readWBWorktreeMetadata(ctx, canonical)
		}()
		wg.Wait()
		for index := range items {
			if wb, ok := wbByPath[filepath.Clean(items[index].Path)]; ok {
				copy := wb
				items[index].WB = &copy
				if !copy.LastCommit.IsZero() {
					items[index].LastCommit = copy.LastCommit
				}
				items[index].Dirty = copy.Dirty
				items[index].StatusRead = !copy.Missing
			}
		}

		p.nav.app.QueueUpdateDraw(func() {
			if loadID != p.loadID.Load() {
				return
			}
			p.loading = false
			p.cancel = nil
			p.items = items
			managed := 0
			for _, item := range items {
				if item.WB != nil && item.WB.HasManifest {
					managed++
				}
			}
			p.status.SetText(fmt.Sprintf("[green]%d Git worktrees[-] · [#7aa2f7]%d WB managed[-] · %s", len(items), managed, tview.Escape(wbStatus)))
			p.renderTable(repoRoot)
		})
	}()
}

func (p *worktreesPanel) stopLoading() {
	if p == nil || p.cancel == nil {
		return
	}
	p.cancel()
	p.cancel = nil
	p.loading = false
}

func (p *worktreesPanel) renderTable(currentPath string) {
	p.table.Clear()
	headers := []string{"Kind", "Branch", "State"}
	for column, title := range headers {
		cell := tview.NewTableCell(title).SetTextColor(tcell.ColorLightSteelBlue).SetSelectable(false)
		p.table.SetCell(0, column, cell)
	}
	selectedRow := 1
	for index, item := range p.items {
		kind := "Git"
		kindColor := tcell.ColorGray
		switch {
		case item.Canonical:
			kind, kindColor = "Clone", tcell.ColorCornflowerBlue
		case item.WB != nil && item.WB.HasManifest:
			kind, kindColor = "WB", tcell.ColorMediumPurple
		}
		branch := item.Branch
		if branch == "" {
			branch = shortSHA(item.Head)
		}
		state := "…"
		stateColor := tcell.ColorGray
		if item.StatusRead {
			state, stateColor = "clean", tcell.ColorGreen
			if item.Dirty {
				state, stateColor = "dirty", tcell.ColorOrange
			}
		}
		if item.Locked {
			state, stateColor = "locked", tcell.ColorYellow
		}
		if item.Prunable {
			state, stateColor = "missing", tcell.ColorRed
		}
		p.table.SetCell(index+1, 0, tview.NewTableCell(kind).SetTextColor(kindColor))
		p.table.SetCell(index+1, 1, tview.NewTableCell(branch).SetExpansion(1))
		p.table.SetCell(index+1, 2, tview.NewTableCell(state).SetTextColor(stateColor))
		if filepath.Clean(item.Path) == filepath.Clean(currentPath) || filepath.Clean(item.Path) == filepath.Clean(p.repoRoot) {
			selectedRow = index + 1
		}
	}
	if len(p.items) > 0 && p.selected {
		p.table.Select(selectedRow, 0)
		p.renderDetail(selectedRow - 1)
	} else {
		p.clearDetail()
	}
}

func (p *worktreesPanel) selectDefaultRow() {
	if p == nil || len(p.items) == 0 {
		return
	}
	if !p.selected {
		p.selected = true
		p.table.Select(1, 0)
	}
	row, _ := p.table.GetSelection()
	p.renderDetail(row - 1)
}

func (p *worktreesPanel) clearDetail() {
	if p == nil || p.detail == nil {
		return
	}
	p.detail.SetText("")
	if p.rows != nil {
		p.rows.ResizeItem(p.detail, 0, 0)
	}
}

func (p *worktreesPanel) renderDetail(index int) {
	if !p.selected || index < 0 || index >= len(p.items) {
		p.clearDetail()
		return
	}
	p.rows.ResizeItem(p.detail, 0, 2)
	item := p.items[index]
	var detail strings.Builder
	fmt.Fprintf(&detail, "[::b]%s[-]\n", tview.Escape(item.Path))
	if item.Canonical {
		detail.WriteString("[cornflowerblue]Canonical clone[-]\n")
	} else {
		detail.WriteString("Git linked worktree\n")
	}
	if item.Head != "" {
		fmt.Fprintf(&detail, "HEAD  %s\n", shortSHA(item.Head))
	}
	if !item.LastCommit.IsZero() {
		fmt.Fprintf(&detail, "Commit %s\n", item.LastCommit.Local().Format("2006-01-02 15:04"))
	}
	if item.WB == nil || !item.WB.HasManifest {
		if !item.Canonical {
			detail.WriteString("\n[gray]Unmanaged by WB[-]")
		}
	} else {
		wb := item.WB
		fmt.Fprintf(&detail, "\n[#7aa2f7::b]WB effort[-] %s\n", tview.Escape(wb.EffortID))
		if wb.ParentEffort != "" {
			fmt.Fprintf(&detail, "Parent %s\n", tview.Escape(wb.ParentEffort))
		}
		if wb.OwnerAgent != "" {
			fmt.Fprintf(&detail, "Owner  %s (%s)\n", tview.Escape(wb.OwnerAgent), tview.Escape(wb.OwnerState))
		} else if wb.OwnerState != "" {
			fmt.Fprintf(&detail, "Owner  %s\n", tview.Escape(wb.OwnerState))
		}
		fmt.Fprintf(&detail, "State  %s · %s\n", tview.Escape(wb.Disposition), tview.Escape(wb.Layout))
	}
	detail.WriteString("\n[gray]Enter open · r refresh · esc close[-]")
	p.detail.SetText(detail.String())
}

func readGitWorktrees(ctx context.Context, repoRoot string) (string, []worktreeInfo, error) {
	canonical, err := canonicalWorktreePath(ctx, repoRoot)
	if err != nil {
		return "", nil, err
	}
	out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", nil, fmt.Errorf("git worktree list: %w", err)
	}
	items := make([]worktreeInfo, 0)
	var item worktreeInfo
	flush := func() {
		if item.Path == "" {
			return
		}
		item.Path = filepath.Clean(item.Path)
		item.Canonical = item.Path == filepath.Clean(canonical)
		items = append(items, item)
		item = worktreeInfo{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			item.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			item.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			item.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "detached":
			item.Detached = true
		case line == "locked" || strings.HasPrefix(line, "locked "):
			item.Locked = true
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			item.Prunable = true
		case line == "":
			flush()
		}
	}
	flush()
	if len(items) == 0 {
		return canonical, nil, errors.New("Git reported no worktrees")
	}
	sortWorktrees(items)
	return canonical, items, nil
}

func canonicalWorktreePath(ctx context.Context, repoRoot string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git common directory: %w", err)
	}
	common := filepath.Clean(strings.TrimSpace(string(out)))
	if filepath.Base(common) != ".git" {
		return "", fmt.Errorf("unsupported Git common directory %s", common)
	}
	return filepath.Dir(common), nil
}

func gitRepositoryRoot(path string) string {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(string(out)))
}

func enrichGitWorktrees(ctx context.Context, items []worktreeInfo) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	for index := range items {
		if items[index].Prunable {
			continue
		}
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			status, err := exec.CommandContext(ctx, "git", "-C", items[index].Path, "status", "--porcelain").Output()
			if err == nil {
				items[index].Dirty = len(status) > 0
				items[index].StatusRead = true
			}
			commit, err := exec.CommandContext(ctx, "git", "-C", items[index].Path, "log", "-1", "--format=%cI").Output()
			if err == nil {
				items[index].LastCommit, _ = time.Parse(time.RFC3339, strings.TrimSpace(string(commit)))
			}
		}(index)
	}
	wg.Wait()
}

func readWBWorktreeMetadata(ctx context.Context, canonical string) (map[string]wbWorktreeInfo, string) {
	metadata := make(map[string]wbWorktreeInfo)
	wb, err := exec.LookPath("wb")
	if err != nil {
		return metadata, "WB not installed; showing raw Git"
	}
	projectsRoot := filepath.Dir(filepath.Dir(canonical))
	command := exec.CommandContext(ctx, wb, "worktree", "orphans", "--format", "json", "--non-interactive", "--projects-root", projectsRoot)
	out, err := command.Output()
	if err != nil {
		return metadata, "WB metadata unavailable"
	}
	var report wbOrphanReport
	if err := json.Unmarshal(out, &report); err != nil {
		return metadata, "WB returned unsupported JSON"
	}
	for _, family := range report.Families {
		for _, worktree := range family.Worktrees {
			if filepath.Clean(worktree.CanonicalDir) == filepath.Clean(canonical) {
				metadata[filepath.Clean(worktree.Path)] = worktree
			}
		}
	}
	return metadata, "WB metadata loaded"
}

func repositoryLabel(repoRoot string) string {
	clean := filepath.Clean(repoRoot)
	return filepath.Base(filepath.Dir(clean)) + "/" + filepath.Base(clean)
}

func shortSHA(value string) string {
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func sortWorktrees(items []worktreeInfo) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Canonical != items[j].Canonical {
			return items[i].Canonical
		}
		return items[i].Branch < items[j].Branch
	})
}
