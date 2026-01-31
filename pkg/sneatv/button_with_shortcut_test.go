package sneatv

import (
	"testing"

	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewButtonWithShortcut(t *testing.T) {
	t.Parallel()
	btn := NewButtonWithShortcut("Save", 's')
	assert.NotNil(t, btn)
	assert.Equal(t, "Save", btn.GetLabel())
	assert.Equal(t, rune('s'), btn.shortcut)
}

func TestButtonWithShortcut_SetShortcutStyle(t *testing.T) {
	t.Parallel()
	btn := NewButtonWithShortcut("Save", 's')
	style := tcell.StyleDefault.Foreground(tcell.ColorRed)
	btn.SetShortcutStyle(style)
	assert.Equal(t, style, btn.shortcutStyle)
}

func TestButtonWithShortcut_Draw(t *testing.T) {
	t.Parallel()
	width := 20
	height := 3
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	btn := NewButtonWithShortcut("Save", 's')
	btn.SetRect(0, 0, width, height)
	btn.Draw(s)

	// The label should be drawn centered.
	// Label "Save" + shortcut "(s)" -> "(s) Save"
	// Total length: 3 (shortcut) + 1 (space) + 4 (Save) = 8
	// Center row is y=1
	line := ttestutils.ReadLine(s, 1, width)
	assert.Contains(t, line, "(s) Save")

	// Test disabled state
	s.Clear()
	btn.SetDisabled(true)
	btn.Draw(s)
	line = ttestutils.ReadLine(s, 1, width)
	assert.Contains(t, line, "(s) Save")
	// We can't easily check colors here without more complex logic,
	// but we verified it doesn't crash and renders the text.

	// Test focus state
	s.Clear()
	btn.SetDisabled(false)
	btn.Focus(func(p tview.Primitive) {})
	btn.Draw(s)
	line = ttestutils.ReadLine(s, 1, width)
	assert.Contains(t, line, "(s) Save")
}

func TestButtonWithShortcut_Draw_ShortWidthFallbackStyle(t *testing.T) {
	t.Parallel()
	width := 4
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	btn := NewButtonWithShortcut("Save", 's')
	btn.SetShortcutStyle(tcell.Style{})
	btn.SetRect(0, 0, width, height)
	btn.Draw(s)

	line := ttestutils.ReadLine(s, 0, width)
	assert.Contains(t, line, "(")
}
