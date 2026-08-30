package charmui

import (
	"charm.land/lipgloss/v2"
	"github.com/strongo/strongo-tui/charm"
)

// styleAdapter is the Filetug layout edge for the provider-owned semantic
// theme. It adds no tokens or mutable theme state: all colours and style
// semantics stay in the dependency-clean strongo-tui Charm module.
type styleAdapter struct {
	theme charm.Theme
}

func newStyleAdapter(theme charm.Theme) styleAdapter {
	return styleAdapter{theme: theme}
}

func (adapter styleAdapter) panel(focused bool, outerWidth, contentHeight int) lipgloss.Style {
	style := adapter.theme.BlurredPanel()
	if focused {
		style = adapter.theme.FocusedPanel()
	}
	contentWidth := maxInt(1, outerWidth-style.GetHorizontalFrameSize())
	return style.Width(contentWidth).Height(contentHeight + style.GetVerticalFrameSize())
}

func (adapter styleAdapter) panelFrameWidth() int {
	return adapter.theme.FocusedPanel().GetHorizontalFrameSize()
}
