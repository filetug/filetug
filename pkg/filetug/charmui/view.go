package charmui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View creates the terminal projection. Child components expose strings in
// Bubbles v2, while the parent owns the Bubble Tea v2 tea.View boundary.
func (m *Model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		view := tea.NewView("Loading Filetug…")
		view.AltScreen = true
		return view
	}
	layout := m.panelLayout()
	if layout.compact {
		return m.compactView()
	}
	treePanel := m.panel("Directories", m.tree.View(), focusTree, layout.treeOuter)
	filesPanel := m.panel("Files", m.filesView(), focusFiles, layout.filesOuter)
	previewPanel := m.panel("Preview", m.preview.View(), focusPreview, layout.previewOuter)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, " ", filesPanel, " ", previewPanel)
	footer := m.footerView()
	content := lipgloss.JoinVertical(lipgloss.Left, m.titleView(), panels, footer)
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m *Model) titleView() string {
	title := fmt.Sprintf("Filetug · %s", m.currentPath)
	return m.styles.theme.Title().Width(maxInt(1, m.width)).Render(title)
}

func (m *Model) panel(title, content string, focus focusPanel, width int) string {
	style := m.styles.panel(m.focus == focus, width, m.panelContentHeight())
	prefixed := m.styles.theme.Title().Render(title) + "\n" + content
	return style.Render(prefixed)
}

func (m *Model) filesView() string {
	switch m.directoryState {
	case "loading":
		return m.styles.theme.Loading().Render("Loading directory…")
	case "empty":
		return m.styles.theme.Empty().Render("Directory is empty")
	case "error":
		return m.styles.theme.Error().Render("Directory error:\n" + m.err)
	}
	if len(m.entries) == 0 {
		return m.styles.theme.Empty().Render("Directory is empty")
	}
	last := minInt(len(m.entries), m.listOffset+m.listHeight())
	lines := make([]string, 0, last-m.listOffset)
	for index := m.listOffset; index < last; index++ {
		entry := m.entries[index]
		marker := "  "
		if index == m.selected {
			marker = "› "
		}
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		row := marker + name
		if index == m.selected {
			row = m.styles.theme.SelectedRow().Render(row)
		} else {
			row = m.styles.theme.UnselectedRow().Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) footerView() string {
	focus := [...]string{"directories", "files", "preview"}[m.focus]
	help := fmt.Sprintf("tab focus: %s · ↑/↓ navigate · enter open · q quit", focus)
	return m.styles.theme.Help().Render(help)
}

type panelLayout struct {
	treeOuter    int
	filesOuter   int
	previewOuter int
	compact      bool
}

func (m *Model) panelLayout() panelLayout {
	// Two one-column gaps and each provider theme panel frame must fit inside
	// the WindowSizeMsg width. Narrow terminals use a compact projection.
	if m.width < 60 || m.height < 8 {
		return panelLayout{compact: true}
	}
	available := m.width - 2
	treeOuter := available / 4
	filesOuter := available / 3
	previewOuter := available - treeOuter - filesOuter
	return panelLayout{
		treeOuter:    treeOuter,
		filesOuter:   filesOuter,
		previewOuter: previewOuter,
	}
}

func (m *Model) compactView() tea.View {
	content := strings.Join([]string{
		"Filetug (compact)",
		m.currentPath,
		m.filesView(),
		"tab focus · q quit",
	}, "\n")
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}
