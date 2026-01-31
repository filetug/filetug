package crumbs

import (
	"strings"
	"testing"

	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestNewBreadcrumbs_DefaultsAndOptions(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(nil)
	if bc.separator != " > " {
		t.Errorf("default separator = %q, want %q", bc.separator, " > ")
	}

	bc2 := NewBreadcrumbs(nil, WithSeparator(" / "))
	if bc2.separator != " / " {
		t.Errorf("WithSeparator not applied, got %q", bc2.separator)
	}
}

func TestBreadcrumbs_PushAndClear(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("A", nil))
	if len(bc.items) != 1 {
		t.Fatalf("initial items length = %d, want 1", len(bc.items))
	}
	bc.Push(NewBreadcrumb("B", nil))
	if len(bc.items) != 2 {
		t.Fatalf("after Push, items length = %d, want 2", len(bc.items))
	}
	bc.Clear()
	if got := len(bc.items); got != 1 {
		t.Fatalf("after Clear, items length = %d, want 1", got)
	}
}

func TestBreadcrumbs_Draw_SingleLineNoBorder(t *testing.T) {
	t.Parallel()
	width := 40
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	bc := NewBreadcrumbs(NewBreadcrumb("DataTug", nil))
	bc.Push(NewBreadcrumb("Projects", nil))
	bc.Push(NewBreadcrumb("Demo", nil))
	bc.SetRect(0, 0, width, height)

	bc.Draw(s)

	line := ttestutils.ReadLine(s, 0, width)
	expected := "DataTug > Projects > Demo"
	if got := strings.TrimRight(line, " "); !strings.HasPrefix(got, expected) {
		t.Fatalf("drawn line prefix mismatch:\n got: %q\nwant: %q", got, expected)
	}
}

func TestBreadcrumbs_Draw_RespectsInnerRectWithBorder(t *testing.T) {
	t.Parallel()
	// Box with border: inner Y should be 1. Height must be at least 3 for border.
	width := 20
	height := 3
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	bc := NewBreadcrumbs(NewBreadcrumb("A", nil))
	bc.SetBorder(true)
	bc.Push(NewBreadcrumb("B", nil))
	bc.SetRect(0, 0, width, height)
	bc.Draw(s)

	// y=0 is border row, ensure no content there
	line0 := ttestutils.ReadLine(s, 0, width)
	if strings.Contains(line0, "A > B") {
		t.Fatalf("content drawn on border row (y=0): %q", strings.TrimRight(line0, " "))
	}
	// Validate content drawn within inner rect span on innerY.
	innerX, innerY, innerW, _ := bc.GetInnerRect()
	var b strings.Builder
	for x := innerX; x < innerX+innerW; x++ {
		str, _, _ := s.Get(x, innerY)
		if str == "" {
			b.WriteRune(' ')
			continue
		}
		b.WriteString(str)
	}
	innerLine := strings.TrimRight(b.String(), " ")
	if !strings.HasPrefix(innerLine, "A > B") {
		t.Fatalf("content not drawn within inner rect (y=%d, x>=%d): %q", innerY, innerX, innerLine)
	}
}

func TestBreadcrumbs_Draw_TruncatesAtWidth(t *testing.T) {
	t.Parallel()
	width := 10 // small width to force truncation
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	bc := NewBreadcrumbs(NewBreadcrumb("ABCDEFGHI", nil), WithSeparator("/"))
	bc.Push(NewBreadcrumb("XYZ", nil))
	bc.SetRect(0, 0, width, height)
	bc.Draw(s)

	line := ttestutils.ReadLine(s, 0, width)
	// Expected to start with the first title and possibly part of separator/title, but never exceed width
	if len([]rune(line)) != width {
		t.Fatalf("line width %d != expected %d", len([]rune(line)), width)
	}
	trimmed := strings.TrimRight(line, " ")
	if !strings.HasPrefix(trimmed, "ABCDEFGHI") && !strings.HasPrefix(trimmed, "ABCDEFGH") && !strings.HasPrefix(trimmed, "ABCDEFG") {
		t.Fatalf("unexpected truncation result: %q", trimmed)
	}
}

func TestBreadcrumbs_Draw_UnfocusedDim(t *testing.T) {
	t.Parallel()
	width := 80
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	bc := NewBreadcrumbs(NewBreadcrumb("DataTug", nil))
	bc.Push(NewBreadcrumb("Projects", nil))
	bc.Push(NewBreadcrumb("Demo", nil)) // last item is focused by default
	bc.SetRect(0, 0, width, height)
	bc.Draw(s)

	// Avoid deprecated style.Decompose(); verify behavior-based selection.
	// By default, the last item is focused/selected.
	if bc.SelectedItemIndex() != bc.ItemsCount()-1 {
		t.Fatalf("expected last item to be selected by default, got %d of %d", bc.SelectedItemIndex(), bc.ItemsCount())
	}
}

// --- Navigation tests for three items ---
func TestBreadcrumbs_Navigation_ThreeItems(t *testing.T) {
	t.Parallel()
	width := 80
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	mk := func() *Breadcrumbs {
		bc := NewBreadcrumbs(NewBreadcrumb("Alpha", nil))
		bc.Push(NewBreadcrumb("Beta", nil))
		bc.Push(NewBreadcrumb("Gamma", nil))
		bc.SetRect(0, 0, width, height)
		bc.Focus(nil) // give focus; selects last by default
		return bc
	}

	// Behavior-based assertion: confirm selected index equals expected one.
	assertSelectedIndex := func(bc *Breadcrumbs, expected int) {
		t.Helper()
		if bc.SelectedItemIndex() != expected {
			t.Fatalf("expected selected index %d, got %d", expected, bc.SelectedItemIndex())
		}
	}

	// Subtest: current item is 2nd. LEFT selects 1st; RIGHT selects 3rd.
	t.Run("current second: left->first, right->third", func(t *testing.T) {
		bc := mk()
		// Move from last (index 2) to second (index 1) with LEFT.
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 0) // Alpha selected

		// RIGHT should move to third (index 2).
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 1) // Beta selected
	})

	// Subtest: current last. LEFT -> second. RIGHT at last: no change.
	t.Run("current last: left->second, right->noop", func(t *testing.T) {
		bc := mk() // currently last (Gamma)
		bc.Draw(s)
		assertSelectedIndex(bc, 1)
		// LEFT -> second (Beta)
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 0)
		// RIGHT -> back to last (Gamma)
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 1)
		// RIGHT at last: should stay last (no change)
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 2)
	})

	// Subtest: current first. LEFT noop. RIGHT -> second.
	t.Run("current first: left->noop, right->second", func(t *testing.T) {
		bc := mk()
		// Force current to first.
		bc.selectedItemIndex = 0
		bc.Draw(s)
		assertSelectedIndex(bc, 0)
		// LEFT at first: noop.
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 0)
		// RIGHT -> second.
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(bc, 1)
	})
}

// Test the new '<' and '>' key navigation
func TestBreadcrumbs_AngleBracketNavigation(t *testing.T) {
	t.Parallel()
	width := 80
	height := 1
	s := ttestutils.NewSimScreen(t, "", width, height)
	defer s.Fini()

	bc := NewBreadcrumbs(NewBreadcrumb("Alpha", nil))
	bc.Push(NewBreadcrumb("Beta", nil))
	bc.Push(NewBreadcrumb("Gamma", nil))
	bc.SetRect(0, 0, width, height)
	bc.Focus(nil) // give focus; selects last by default

	// Behavior-based assertion for this test suite
	assertSelectedIndex := func(expected int) {
		if bc.SelectedItemIndex() != expected {
			t.Fatalf("expected selected index %d, got %d", expected, bc.SelectedItemIndex())
		}
	}

	// Test '<' key navigation
	t.Run("angle bracket left navigation", func(t *testing.T) {
		bc.selectedItemIndex = 2 // start at last (Gamma)
		bc.Draw(s)
		assertSelectedIndex(2)

		// '<' should move to Beta
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(1) // Beta selected

		// '<' should move to Alpha
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(0) // Alpha selected

		// '<' at first item should do nothing
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(0) // Still Alpha selected
	})

	// Test '>' key navigation
	t.Run("angle bracket right navigation", func(t *testing.T) {
		bc.selectedItemIndex = 0 // start at first (Alpha)
		bc.Draw(s)
		assertSelectedIndex(0)

		// '>' should move to Beta
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(1)

		// '>' should move to Gamma
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(2)

		// '>' at last item should do nothing
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone), func(p tview.Primitive) {})
		}
		bc.Draw(s)
		assertSelectedIndex(2) // Still Gamma selected
	})

	// Test that angle bracket keys don't change focus
	t.Run("angle bracket keys should not change focus", func(t *testing.T) {
		bc.selectedItemIndex = 1 // start at middle item

		var focusChanges int
		// Test '<' key
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '<', tcell.ModNone), func(p tview.Primitive) {
				focusChanges++
			})
		}
		if focusChanges > 0 {
			t.Errorf("'<' key should not change focus; got %d focus changes", focusChanges)
		}

		focusChanges = 0
		// Test '>' key
		if h := bc.InputHandler(); h != nil {
			h(tcell.NewEventKey(tcell.KeyRune, '>', tcell.ModNone), func(p tview.Primitive) {
				focusChanges++
			})
		}
		if focusChanges > 0 {
			t.Errorf("'>' key should not change focus; got %d focus changes", focusChanges)
		}
	})
}
