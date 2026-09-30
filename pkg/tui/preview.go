package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// attrRows is the number of rows above the preview text: size, modified and a
// rule.
const attrRows = 3

// previewPane shows the attributes and the text preview of one entry.
type previewPane struct {
	text     widgets.TextPane
	title    string
	size     string
	modified string
	loading  bool
	// summary, when not nil, is shown instead of the text: the entry is a
	// directory.
	summary *summaryPane
	focused bool
	w, h    int
}

func newPreviewPane() previewPane {
	return previewPane{text: widgets.NewTextPane("preview")}
}

// Title is the name of the previewed entry.
func (p previewPane) Title() string { return p.title }

// SetSize gives the pane its size.
func (p *previewPane) SetSize(w, h int) {
	p.w, p.h = w, h
	p.text.SetSize(w, max(h-attrRows, 0))
	if p.summary != nil {
		p.summary.SetSize(w, max(h-attrRows, 0))
	}
}

// Focus and Blur move the focus into and out of the text or the summary.
func (p *previewPane) Focus() { p.setFocus(true) }
func (p *previewPane) Blur()  { p.setFocus(false) }

func (p *previewPane) setFocus(focused bool) {
	p.focused = focused
	if focused {
		p.text.Focus()
	} else {
		p.text.Blur()
	}
	if p.summary != nil {
		p.summary.SetFocused(focused)
	}
}

// Loading shows that a preview is being built.
func (p *previewPane) Loading(title string) {
	p.title, p.size, p.modified, p.loading = title, "", "", true
	p.summary = nil
	p.text.SetTextColor(theme.MutedColor())
	p.text.SetContent("Loading…")
}

// Show displays a finished preview.
func (p *previewPane) Show(msg previewMsg) {
	p.loading = false
	p.title, p.size, p.modified = msg.Title, msg.Size, msg.Modified
	p.summary = nil
	if msg.Summary != nil {
		p.summary = newSummaryPane(msg.Summary)
		p.summary.SetSize(p.w, max(p.h-attrRows, 0))
		p.summary.SetFocused(p.focused)
		p.text.SetContent("")
		return
	}
	if msg.Err != nil {
		p.text.SetTextColor(theme.ErrorColor())
		p.text.SetContent(msg.Err.Error())
		return
	}
	p.text.SetTextColor(nil)
	p.text.SetContent(msg.Body)
}

// Text is the text being shown.
func (p previewPane) Text() string { return p.text.Content() }

// Update forwards keys and the wheel to the text.
func (p previewPane) Update(msg tea.Msg) (previewPane, tea.Cmd) {
	if p.summary != nil {
		return p, p.summary.Update(msg)
	}
	var cmd tea.Cmd
	p.text, cmd = p.text.Update(msg)
	return p, cmd
}

// AtEdge implements widgets.Boundary.
func (p previewPane) AtEdge(dir widgets.Direction) bool {
	if p.summary != nil {
		return p.summary.AtEdge(dir)
	}
	return p.text.AtEdge(dir)
}

// View draws the attributes above the text.
func (p previewPane) View() string {
	label := lipgloss.NewStyle().Foreground(theme.MutedColor())
	rule := label.Render(strings.Repeat("─", max(p.w, 0)))
	head := []string{
		label.Render("Size     ") + " " + p.size,
		label.Render("Modified") + " " + p.modified,
		rule,
	}
	body := p.text.View()
	if p.summary != nil {
		body = p.summary.View()
	}
	return widgets.Fit(strings.Join(head, "\n")+"\n"+body, p.w, p.h)
}
