package sneatv

import (
	"testing"

	"github.com/datatug/filetug/pkg/sneatv/ttestutils"
	"github.com/rivo/tview"
)

func TestNewBoxed(t *testing.T) {
	inner := tview.NewBox().SetTitle("Inner Title")
	boxed := NewBoxed(inner,
		WithLeftBorder(1, 0),
		WithRightBorder(1, 0),
		WithTabs(&PanelTab{Title: "Tab 1", Hotkey: '1', Checked: true}),
	)

	if boxed == nil {
		t.Fatal("Expected NewBoxed to return a non-nil value")
	}
}

func TestBoxed_Draw(t *testing.T) {
	screen := ttestutils.NewSimScreen(t, "UTF-8", 40, 10)

	inner := tview.NewBox().SetTitle("Inner Title")
	boxed := NewBoxed(inner,
		WithLeftBorder(1, 0),
		WithRightBorder(1, 0),
	)
	boxed.SetRect(0, 0, 40, 10)

	// Test blurred state
	boxed.Blur()
	boxed.Draw(screen)
	screen.Show()

	// Test focused state
	boxed.Focus(func(p tview.Primitive) {})
	boxed.Draw(screen)
	screen.Show()

	// Test with tabs
	boxedWithTabs := NewBoxed(inner,
		WithTabs(&PanelTab{Title: "Tab 1", Hotkey: '1', Checked: true}),
		WithTabs(&PanelTab{Title: "Tab 2", Hotkey: '2', Checked: false}),
	)
	boxedWithTabs.SetRect(0, 0, 40, 10)
	boxedWithTabs.Focus(func(p tview.Primitive) {})
	boxedWithTabs.Draw(screen)
	screen.Show()
}
