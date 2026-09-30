package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/filetug/masks"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// panelKind names a panel that can take the place of the preview.
type panelKind int

const (
	panelNew panelKind = iota
	panelScripts
	panelGenerator
	panelMasks
)

// openPanelMsg puts a panel in place of the preview.
type openPanelMsg struct{ Kind panelKind }

// panelClosedMsg removes the panel; focus goes to To.
type panelClosedMsg struct{ To nav.FocusTo }

// createEntryMsg asks to create a directory or an empty file in the current
// directory.
type createEntryMsg struct {
	Dir  bool
	Name string
}

// panel is something shown in place of the preview: a form, a list, a table.
// Panels are pointers: they change in place and the browser only replaces them.
type panel interface {
	Title() string
	SetSize(w, h int)
	SetFocused(focused bool)
	Update(msg tea.Msg) tea.Cmd
	View() string
}

// newPanelFor creates the panel of a kind.
func newPanelFor(kind panelKind) panel {
	switch kind {
	case panelScripts:
		return newScriptsPanel()
	case panelGenerator:
		return newGeneratorPanel()
	case panelMasks:
		return newMasksPanel()
	}
	return newNewPanel()
}

// closePanel closes the panel and sends focus to a zone.
func closePanel(to nav.FocusTo) tea.Cmd { return widgets.Emit(panelClosedMsg{To: to}) }

// newEntryPanel asks for a name and creates a directory or a file (F7).
type newEntryPanel struct {
	form widgets.Form
}

func newNewPanel() *newEntryPanel {
	return &newEntryPanel{form: widgets.NewForm("new",
		[]widgets.Field{
			{ID: "name", Label: "Name", Kind: widgets.TextField},
			{ID: "hint", Kind: widgets.StaticField, Text: "Tab: navigate  •  Enter: create file  •  Esc: cancel"},
		},
		[]widgets.FormButton{
			{ID: "dir", Label: "Create directory", Role: widgets.ActionRole},
			{ID: "file", Label: "Create file", Role: widgets.ActionRole},
		})}
}

func (p *newEntryPanel) Title() string     { return "New" }
func (p *newEntryPanel) SetSize(w, h int)  { p.form.SetSize(w, h) }
func (p *newEntryPanel) View() string      { return p.form.View() }
func (p *newEntryPanel) SetFocused(f bool) { p.setFocus(f) }
func (p *newEntryPanel) setFocus(f bool) {
	if f {
		p.form.Focus()
		return
	}
	p.form.Blur()
}

// Update turns the form's messages into entries to create. Enter in the name
// field creates a file.
func (p *newEntryPanel) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && p.form.Editing() {
		return widgets.Emit(createEntryMsg{Name: p.form.Value("name")})
	}
	var cmd tea.Cmd
	p.form, cmd = p.form.Update(msg)
	switch m := msg.(type) {
	case widgets.CancelMsg:
		if m.ID == "new" {
			return closePanel(nav.FocusToMenu)
		}
	case widgets.ButtonPressedMsg:
		if m.ID == "new" {
			return widgets.Emit(createEntryMsg{Dir: m.ButtonID == "dir", Name: m.Values["name"]})
		}
	}
	return cmd
}

var _ panel = (*newEntryPanel)(nil)

// scriptsPanel lists the scripts (F10).
type scriptsPanel struct {
	list widgets.List
}

func newScriptsPanel() *scriptsPanel {
	return &scriptsPanel{list: widgets.NewList("scripts",
		widgets.MenuItem{ID: "nested", Label: "Nested Dirs Generator", Shortcut: '1'})}
}

func (p *scriptsPanel) Title() string    { return "Scripts" }
func (p *scriptsPanel) SetSize(w, h int) { p.list.SetSize(w, h) }
func (p *scriptsPanel) View() string     { return p.list.View() }
func (p *scriptsPanel) SetFocused(f bool) {
	if f {
		p.list.Focus()
		return
	}
	p.list.Blur()
}

func (p *scriptsPanel) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "esc" {
		return closePanel(nav.FocusToContent)
	}
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	if sel, ok := msg.(widgets.ItemSelectedMsg); ok && sel.ID == "scripts" {
		return widgets.Emit(openPanelMsg{Kind: panelGenerator})
	}
	return cmd
}

var _ panel = (*scriptsPanel)(nil)

// generatorPanel is the form of the nested directories generator.
type generatorPanel struct {
	form widgets.Form
}

func newGeneratorPanel() *generatorPanel {
	return &generatorPanel{form: widgets.NewForm("generator",
		[]widgets.Field{
			{ID: "depth", Label: "Depth", Kind: widgets.TextField, Value: "20"},
			{ID: "subdirs", Label: "SubDirs", Kind: widgets.TextField, Value: "10"},
			{ID: "files", Label: "FilesPerDir", Kind: widgets.TextField, Value: "10"},
			{ID: "size", Label: "File Size (bytes)", Kind: widgets.TextField, Value: "1024"},
		},
		[]widgets.FormButton{
			{ID: "generate", Label: "Generate", Role: widgets.ActionRole},
			{ID: "cancel", Label: "Cancel", Role: widgets.CancelRole},
		})}
}

func (p *generatorPanel) Title() string    { return "Nested Dirs Generator" }
func (p *generatorPanel) SetSize(w, h int) { p.form.SetSize(w, h) }
func (p *generatorPanel) View() string     { return p.form.View() }
func (p *generatorPanel) SetFocused(f bool) {
	if f {
		p.form.Focus()
		return
	}
	p.form.Blur()
}

// Update closes the panel on Cancel and Esc; Generate does nothing yet, as in
// the tview version.
func (p *generatorPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.form, cmd = p.form.Update(msg)
	if m, ok := msg.(widgets.CancelMsg); ok && m.ID == "generator" {
		return closePanel(nav.FocusToContent)
	}
	return cmd
}

var _ panel = (*generatorPanel)(nil)

// masksPanel lists the named masks (Alt+M).
type masksPanel struct {
	grid *grid.Model
}

var maskColumns = []grid.Column{{Name: "Mask"}, {Name: "CurrDir", Numeric: true}, {Name: "SubDirs", Numeric: true}}

func newMasksPanel() *masksPanel {
	var rows []grid.Row
	for _, m := range masks.BuiltIn() {
		rows = append(rows, grid.Row{Key: m.Name, Values: []any{m.Name, "...", "..."}})
	}
	return &masksPanel{grid: grid.New(maskColumns, rows,
		grid.WithID("masks"), grid.WithRowSelection(), grid.WithoutFrame(), grid.WithoutSwitcher(),
		grid.WithFilterDisabled(), grid.WithStyle(grid.StyleMinimal),
		grid.WithFooterHook(func(*grid.Model, string) string { return "" }))}
}

func (p *masksPanel) Title() string     { return "Masks" }
func (p *masksPanel) SetSize(w, h int)  { p.grid.SetSize(w, h) }
func (p *masksPanel) SetFocused(f bool) { p.grid.SetFocused(f) }
func (p *masksPanel) View() string      { return p.grid.View(p.grid.Width(), p.grid.Focused()) }

func (p *masksPanel) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && key.Matches(k, key.NewBinding(key.WithKeys("esc"))) {
		return closePanel(nav.FocusToContent)
	}
	_, cmd := p.grid.Update(msg)
	return cmd
}

var _ panel = (*masksPanel)(nil)
