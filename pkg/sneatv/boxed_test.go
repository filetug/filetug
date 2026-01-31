package sneatv

import (
	"strings"
	"testing"

	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/require"
)

func TestNewBoxed(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestBoxed_DrawFooter(t *testing.T) {
	t.Parallel()
	screen := ttestutils.NewSimScreen(t, "UTF-8", 20, 5)

	inner := tview.NewBox().SetTitle("Inner Title")
	boxed := NewBoxed(inner)
	boxed.SetRect(0, 0, 20, 5)
	boxed.Blur()

	boxed.Draw(screen)
	screen.Show()
	line := ttestutils.ReadLine(screen, 4, 20)
	require.Equal(t, strings.Repeat("─", 20), line)

	footer := tview.NewTextView().SetText("FOOT")
	boxedWithFooter := NewBoxed(inner, WithFooter(footer))
	boxedWithFooter.SetRect(0, 0, 20, 5)
	boxedWithFooter.Blur()

	boxedWithFooter.Draw(screen)
	screen.Show()
	line = ttestutils.ReadLine(screen, 4, 20)
	require.Equal(t, "───────┤FOOT├───────", line)
}

func TestBoxed_DrawFooterRerender(t *testing.T) {
	t.Parallel()
	screen := ttestutils.NewSimScreen(t, "UTF-8", 20, 5)

	footer := tview.NewTextView().SetText("FOOT")
	boxed := NewBoxed(tview.NewBox(), WithFooter(footer))
	boxed.SetRect(0, 0, 20, 5)
	boxed.Blur()

	boxed.Draw(screen)
	screen.Show()
	line := ttestutils.ReadLine(screen, 4, 20)
	require.Equal(t, "───────┤FOOT├───────", line)

	footer.SetText("NEW")
	boxed.Draw(screen)
	screen.Show()
	line = ttestutils.ReadLine(screen, 4, 20)
	require.Equal(t, "───────┤NEW├────────", line)
}

func TestWithFooter(t *testing.T) {
	t.Parallel()
	footer := tview.NewTextView().SetText("Footer")
	boxed := NewBoxed(tview.NewBox(), WithFooter(footer))
	require.Equal(t, footer, boxed.options.footer)
}

func TestBorderPrimitiveWidth(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, borderPrimitiveWidth(nil))

	box := tview.NewBox()
	box.SetRect(0, 0, 0, 0)
	require.Equal(t, 0, borderPrimitiveWidth(box))

	box.SetRect(0, 0, 6, 1)
	require.Equal(t, 6, borderPrimitiveWidth(box))

	textFooter := tview.NewTextView().SetText("Hello\nWorld")
	require.Equal(t, 5, borderPrimitiveWidth(textFooter))
}
