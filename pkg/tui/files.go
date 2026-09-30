package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// filesID names the file list's grid in its messages.
const filesID = "files"

// filesPane is the file list: a lazy grid of the entries of one directory.
type filesPane struct {
	grid *grid.Model
	// state is shared with the grid's hooks, which outlive any copy of the pane.
	state *filesState
	err   error
	w, h  int
}

// filesState is what the grid's hooks read and write.
type filesState struct {
	rows  *fileRows
	marks map[string]bool
}

func newFilesPane() filesPane {
	st := &filesState{marks: map[string]bool{}}
	st.rows = newFileRows(nil, false, st.marks)
	p := filesPane{state: st}
	p.grid = grid.New(fileColumns(), nil,
		grid.WithID(filesID),
		grid.WithRowSource(st.rows, fileColumns()),
		grid.WithRowSelection(),
		grid.WithoutFrame(),
		grid.WithoutSwitcher(),
		grid.WithFilterDisabled(),
		grid.WithStyle(grid.StyleMinimal),
		grid.WithCellStyle(cellStyle),
		grid.WithKeyHandler(st.claimKey),
		grid.WithFooterHook(st.footer),
	)
	return p
}

// footer summarises the list: how many entries it has and how many are selected.
func (s *filesState) footer(_ *grid.Model, _ string) string {
	n := len(s.rows.visible)
	text := fmt.Sprintf("%d items", n)
	if n == 1 {
		text = "1 item"
	}
	if len(s.marks) > 0 {
		text += fmt.Sprintf(" · %d selected", len(s.marks))
	}
	return text
}

// cellStyle colours the name by extension and marks failed rows.
func cellStyle(row grid.Row, column int, _ any) lipgloss.Style {
	ref, ok := row.Ref.(rowRef)
	if !ok {
		return lipgloss.NewStyle()
	}
	switch {
	case ref.Err != nil && column == modifiedColumn:
		return lipgloss.NewStyle().Foreground(theme.ErrorColor())
	case column == nameColumn && ref.IsDir:
		return lipgloss.NewStyle().Foreground(theme.FocusColor())
	case column == nameColumn:
		return lipgloss.NewStyle().Foreground(fileNameColor(ref.Entry.Name()))
	}
	return lipgloss.NewStyle().Foreground(theme.MutedColor())
}

// claimKey handles the keys of the file list that the grid must not see: Space
// marks the entry, and the global shortcuts go to the root and home directory.
func (s *filesState) claimKey(m *grid.Model, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "space":
		if row, ok := m.CurrentRow(); ok {
			s.toggleMark(row)
			m.Refresh()
		}
		return nil, true
	case "/":
		return goDir("/"), true
	case "`":
		return goDir("~"), true
	}
	return nil, false
}

// toggleMark selects or deselects the entry of a row.
func (s *filesState) toggleMark(row grid.Row) {
	ref, ok := row.Ref.(rowRef)
	if !ok || ref.Parent {
		return
	}
	full := ref.Entry.FullName()
	if s.marks[full] {
		delete(s.marks, full)
		return
	}
	s.marks[full] = true
}

// SetSize gives the pane its size.
func (p *filesPane) SetSize(w, h int) {
	p.w, p.h = w, h
	p.grid.SetSize(w, h)
}

// SetDir shows the entries of a directory and selects the entry called
// selectName if there is one.
func (p *filesPane) SetDir(dir *files.DirContext, showDirs bool, selectName string) {
	clear(p.state.marks)
	p.err = nil
	p.state.rows = newFileRows(dir, showDirs, p.state.marks)
	p.grid.SetRowSource(p.state.rows, fileColumns())
	if i := p.state.rows.IndexOf(selectName); i >= 0 && selectName != "" {
		p.grid.SelectRow(i)
	}
}

// SetGitText shows the git status of an entry after its name.
func (p *filesPane) SetGitText(fullPath, text string) {
	if p.state.rows.SetGitText(fullPath, text) {
		p.grid.Refresh()
	}
}

// SetExtFilter shows only the files with these extensions; none shows all
// files.
func (p *filesPane) SetExtFilter(extensions []string) {
	filter := p.state.rows.filter
	filter.Extensions = extensions
	p.state.rows.SetFilter(filter)
	p.grid.Refresh()
}

// SetError shows why a directory could not be read.
func (p *filesPane) SetError(err error) {
	p.err = err
	p.state.rows = newFileRows(nil, false, p.state.marks)
	p.grid.SetRowSource(p.state.rows, fileColumns())
}

// Current is the entry under the cursor.
func (p filesPane) Current() files.EntryWithDirPath {
	row, ok := p.grid.CurrentRow()
	if !ok {
		return nil
	}
	ref, _ := row.Ref.(rowRef)
	return ref.Entry
}

// CurrentRef is the reference of the row under the cursor.
func (p filesPane) CurrentRef() (rowRef, bool) {
	row, ok := p.grid.CurrentRow()
	if !ok {
		return rowRef{}, false
	}
	ref, ok := row.Ref.(rowRef)
	return ref, ok
}

// Update forwards a message to the grid.
func (p filesPane) Update(msg tea.Msg) (filesPane, tea.Cmd) {
	var cmd tea.Cmd
	p.grid, cmd = p.grid.Update(msg)
	return p, cmd
}

// AtEdge implements widgets.Boundary.
func (p filesPane) AtEdge(dir widgets.Direction) bool { return p.grid.AtEdge(dir) }

// View draws the list, or the reason there is none.
func (p filesPane) View(focused bool) string {
	switch {
	case p.err != nil:
		return widgets.Fit(lipgloss.NewStyle().Foreground(theme.ErrorColor()).Render(dirEmoji+" "+p.err.Error()), p.w, p.h)
	case p.state.rows.Len() == 0:
		return widgets.Fit(lipgloss.NewStyle().Foreground(theme.MutedColor()).Italic(true).Render("No entries"), p.w, p.h)
	}
	return widgets.Fit(p.grid.View(p.w, focused), p.w, p.h)
}
