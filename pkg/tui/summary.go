package tui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// Summary tabs.
const (
	tabTypes = iota
	tabGit
)

const (
	typesID = "types"
	gitID   = "gitstatus"
)

// extFilterMsg narrows the file list to files with these extensions; no
// extensions shows all files again.
type extFilterMsg struct{ Extensions []string }

// focusFilesMsg moves the focus back to the file list.
type focusFilesMsg struct{}

// typeRef is what a row of the file types table stands for.
type typeRef struct {
	Extensions []string
	Group      bool
}

// summaryPane shows what a directory holds: the file types with their counts
// and sizes, and, inside a repository, the files git reports as changed.
type summaryPane struct {
	data    *dirSummary
	tabs    widgets.Tabs
	types   *grid.Model
	changes *grid.Model
	active  int
	focused bool
	w, h    int
}

func newSummaryPane(data *dirSummary) *summaryPane {
	p := &summaryPane{data: data}
	tabs := []widgets.Tab{{ID: typesID, Title: "File types"}}
	if data.Git != nil {
		tabs = append(tabs, widgets.Tab{ID: gitID, Title: "Git"})
	}
	p.tabs = widgets.NewTabs("summary", widgets.UnderlineTabsStyle, tabs...)
	p.types = newSummaryGrid(typesID, []grid.Column{{Name: "Type"}, {Name: "Files", Numeric: true}, {Name: "Size", Numeric: true}}, typeCellStyle)
	p.types.SetData(p.types.Columns(), typeRows(data.Groups))
	p.changes = newSummaryGrid(gitID, []grid.Column{{Name: "File", MaxWidth: 50}, {Name: "Change"}}, changeCellStyle)
	p.setGit(data.Git)
	if data.Git != nil && len(data.Git.Entries) > 0 {
		p.activate(tabGit)
	}
	return p
}

func newSummaryGrid(id string, columns []grid.Column, style grid.CellStyleFunc) *grid.Model {
	return grid.New(columns, nil, grid.WithID(id), grid.WithRowSelection(), grid.WithoutFrame(), grid.WithoutSwitcher(),
		grid.WithFilterDisabled(), grid.WithStyle(grid.StyleMinimal), grid.WithCellStyle(style),
		grid.WithFooterHook(func(*grid.Model, string) string { return "" }))
}

// countText is "1 file" or "N files".
func countText(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

// typeRows are the rows of the file types table: a row for each kind of file
// and, below it, one for each of its extensions.
func typeRows(groups []*extGroup) []grid.Row {
	var rows []grid.Row
	for _, g := range groups {
		count, size := "", ""
		if len(g.Exts) > 1 {
			count, size = countText(g.Count), fsutils.GetSizeShortText(g.Size)
		}
		rows = append(rows, grid.Row{Key: "group:" + g.ID, Values: []any{"▼ " + g.Title, count, size},
			Ref: typeRef{Extensions: g.Extensions(), Group: true}})
		for _, e := range g.Exts {
			name := "  *" + e.ID
			if e.ID == "" {
				name = "  <no extension>"
			}
			rows = append(rows, grid.Row{Key: "ext:" + e.ID, Values: []any{name, countText(e.Count), fsutils.GetSizeShortText(e.Size)},
				Ref: typeRef{Extensions: []string{e.ID}}})
		}
	}
	return rows
}

// typeCellStyle marks the kinds of file and colours sizes by magnitude.
func typeCellStyle(row grid.Row, column int, value any) lipgloss.Style {
	ref, _ := row.Ref.(typeRef)
	style := lipgloss.NewStyle()
	if ref.Group {
		style = style.Bold(true)
	}
	if column == 2 {
		if text, ok := value.(string); ok && text != "" {
			style = style.Foreground(sizeColor(text))
		}
	}
	return style
}

// sizeColor colours a size text: from muted for bytes to red for terabytes.
func sizeColor(text string) color.Color {
	switch {
	case strings.HasSuffix(text, "TB"):
		return theme.ErrorColor()
	case strings.HasSuffix(text, "GB"):
		return theme.AccentColor()
	case strings.HasSuffix(text, "MB"):
		return adaptive("#3BD17F", "#1B7F46")
	case strings.HasSuffix(text, "KB"):
		return theme.TextColor()
	}
	return theme.MutedColor()
}

// setGit fills the git table from a status.
func (p *summaryPane) setGit(status *gitDirStatus) {
	p.data.Git = status
	var rows []grid.Row
	if status != nil {
		for _, e := range status.Entries {
			mark := " "
			if e.Staged {
				mark = "✓"
			}
			rows = append(rows, grid.Row{Key: e.FullPath, Values: []any{mark + " " + e.DisplayName, e.Badge.Text + ":" + e.Badge.Label}, Ref: e})
		}
	}
	p.changes.SetData(p.changes.Columns(), rows)
}

// changeCellStyle colours the change of a file by its kind.
func changeCellStyle(row grid.Row, column int, _ any) lipgloss.Style {
	entry, ok := row.Ref.(gitEntry)
	if !ok || column != 1 {
		return lipgloss.NewStyle()
	}
	switch entry.Badge.Text {
	case "A":
		return lipgloss.NewStyle().Foreground(adaptive("#3BD17F", "#1B7F46"))
	case "D":
		return lipgloss.NewStyle().Foreground(theme.ErrorColor())
	case "M":
		return lipgloss.NewStyle().Foreground(theme.AccentColor())
	}
	return lipgloss.NewStyle().Foreground(theme.MutedColor())
}

// SetSize gives the pane its size.
func (p *summaryPane) SetSize(w, h int) {
	p.w, p.h = w, h
	p.tabs.SetSize(w, 1)
	p.types.SetSize(w, max(h-1, 0))
	p.changes.SetSize(w, max(h-1, 0))
}

// SetFocused tells the pane whether it has the focus.
func (p *summaryPane) SetFocused(focused bool) {
	p.focused = focused
	if focused {
		p.tabs.Focus()
	} else {
		p.tabs.Blur()
	}
	p.types.SetFocused(focused)
	p.changes.SetFocused(focused)
}

// activate shows a tab.
func (p *summaryPane) activate(tab int) {
	p.active = tab
	p.tabs.SetActive(tab)
}

// body is the table of the active tab.
func (p *summaryPane) body() *grid.Model {
	if p.active == tabGit {
		return p.changes
	}
	return p.types
}

// Filter is the extension filter for the row under the cursor of the file types
// table: a file list that shows everything when the cursor is elsewhere.
func (p *summaryPane) Filter() tea.Cmd {
	if p.active != tabTypes {
		return nil
	}
	row, ok := p.types.CurrentRow()
	if !ok {
		return widgets.Emit(extFilterMsg{})
	}
	ref, _ := row.Ref.(typeRef)
	return widgets.Emit(extFilterMsg{Extensions: ref.Extensions})
}

// Update handles keys: Left and Right move between the tabs, and Left on the
// first tab returns to the file list; Space stages or unstages the file under
// the cursor of the git tab; the rest is the table's.
func (p *summaryPane) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "left":
		if p.active == tabTypes {
			return widgets.Emit(focusFilesMsg{})
		}
		p.activate(tabTypes)
		return p.Filter()
	case "right":
		if p.data.Git != nil && p.active == tabTypes {
			p.activate(tabGit)
		}
		return nil
	case "space":
		return p.toggleStage()
	}
	_, cmd := p.body().Update(msg)
	return cmd
}

// toggleStage stages the changed file under the cursor, or unstages it.
func (p *summaryPane) toggleStage() tea.Cmd {
	if p.active != tabGit {
		return nil
	}
	row, ok := p.changes.CurrentRow()
	if !ok {
		return nil
	}
	entry, ok := row.Ref.(gitEntry)
	if !ok {
		return nil
	}
	return stageCmd(p.data.Path, entry)
}

// GitLoaded shows a fresh git status of the directory.
func (p *summaryPane) GitLoaded(msg gitDirLoadedMsg) {
	if msg.Path == p.data.Path {
		p.setGit(msg.Status)
	}
}

// Selection is the grid message a selection change of the file types table
// reports, turned into the matching filter.
func (p *summaryPane) Selection() tea.Cmd { return p.Filter() }

// AtEdge implements widgets.Boundary for the table of the active tab.
func (p *summaryPane) AtEdge(dir widgets.Direction) bool {
	if dir == widgets.Left || dir == widgets.Right {
		return false
	}
	return p.body().AtEdge(dir)
}

// View draws the tab strip and the active table, or the reason there is none.
func (p *summaryPane) View() string {
	body := p.body().View(p.w, p.focused)
	if p.active == tabGit {
		body = p.gitView()
	}
	return widgets.Fit(p.tabs.View()+"\n"+body, p.w, p.h)
}

// gitView is the git table, or a line that says why it is empty.
func (p *summaryPane) gitView() string {
	muted := lipgloss.NewStyle().Foreground(theme.MutedColor())
	switch {
	case p.data.Git == nil:
		return muted.Render("Not a git repository")
	case p.data.Git.Err != nil:
		return lipgloss.NewStyle().Foreground(theme.ErrorColor()).Render(p.data.Git.Err.Error())
	case len(p.data.Git.Entries) == 0:
		return muted.Render("No changes")
	}
	return p.changes.View(p.w, p.focused)
}
