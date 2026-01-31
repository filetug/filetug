package crumbs

import (
	"testing"

	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBreadcrumbs_Draw_EdgeCases(t *testing.T) {
	t.Parallel()
	// width <= 0
	bc := NewBreadcrumbs(NewBreadcrumb("A", nil))
	bc.SetRect(0, 0, 0, 1)
	s := tcell.NewSimulationScreen("UTF-8")
	_ = s.Init()
	bc.Draw(s) // should return early

	// URL detection and default color
	width := 40
	height := 1
	s2 := ttestutils.NewSimScreen(t, "", width, height)
	defer s2.Fini()

	bc2 := NewBreadcrumbs(NewBreadcrumb("http://example.com", nil)) // URL
	bc2.Push(NewBreadcrumb("plain", nil))                           // Default color (0)
	bc2.SetRect(0, 0, width, height)
	bc2.Focus(func(p tview.Primitive) {}) // Focus it to trigger yellow highlight code path
	bc2.Draw(s2)

	// Truncation case
	bc3 := NewBreadcrumbs(NewBreadcrumb("verylongname", nil))
	bc3.SetRect(0, 0, 5, 1)
	bc3.Draw(s2)

	line := ttestutils.ReadLine(s2, 0, width)

	if !testing.Short() {
		_ = line
		//t.Logf("Line: %q", line)
	}
}

func TestBreadcrumbs_MouseHandler_EdgeCases(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Alpha", nil))
	bc.SetRect(0, 0, 20, 1)

	s := tcell.NewSimulationScreen("UTF-8")
	_ = s.Init()
	defer s.Fini()

	handler := bc.MouseHandler()

	// Other mouse actions (e.g., Move)
	ev := tcell.NewEventMouse(1, 0, tcell.ButtonNone, 0)
	consumed, _ := handler(tview.MouseMove, ev, nil)
	if consumed {
		t.Error("MouseMove should not be consumed")
	}

	// Out of inner rect (above)
	ev = tcell.NewEventMouse(1, -1, tcell.Button1, 0)
	consumed, _ = handler(tview.MouseLeftClick, ev, nil)
	if consumed {
		t.Error("Click outside inner rect should not be consumed")
	}

	// Click on separator or empty space
	// "Alpha" is 5 chars. Separator " > " is 3 chars.
	// Index 0-4: Alpha
	// Index 5-7: Separator
	bc.Push(NewBreadcrumb("Beta", nil))
	bc.Draw(s) // Update internal state if any (though Box doesn't store layout)

	// Click on separator (index 6)
	ev = tcell.NewEventMouse(6, 0, tcell.Button1, 0)
	var focused tview.Primitive
	consumed, _ = handler(tview.MouseLeftClick, ev, func(p tview.Primitive) { focused = p })
	if !consumed {
		t.Error("Click on separator should be consumed")
	}
	if focused != bc {
		t.Errorf("Expected breadcrumbs to be focused, got %v", focused)
	}

	// Click on empty space (not beyond Beta yet, since Beta is at 5+3=8. Alpha(5) + " > "(3) + Beta(4) = 12)
	// Let's click at 15
	ev = tcell.NewEventMouse(15, 0, tcell.Button1, 0)
	consumed, _ = handler(tview.MouseLeftClick, ev, func(p tview.Primitive) { focused = p })
	if !consumed {
		t.Error("Click on empty space should be consumed")
	}

	// MouseLeftDown on an item (should NOT call action)
	bc.Push(NewBreadcrumb("Gamma", func() error { return nil }))
	bc.Draw(s)

	// Gamma starts at 12 (Alpha=5, " > "=3, Beta=4). Next separator " > " is 3 chars. Gamma is at 15.
	// Beta is at 8. Beta length is 4. Beta ends at 11.
	// Click on Beta
	ev = tcell.NewEventMouse(10, 0, tcell.Button1, 0)
	consumed, _ = handler(tview.MouseLeftDown, ev, func(p tview.Primitive) { focused = p })
	if !consumed {
		t.Error("MouseLeftDown on item should be consumed")
	}
	if bc.selectedItemIndex != 1 {
		t.Errorf("Expected Beta to be selected, got %d", bc.selectedItemIndex)
	}

	// Test cursorX >= maxX in MouseHandler
	bcSmall := NewBreadcrumbs(NewBreadcrumb("A", nil))
	bcSmall.Push(NewBreadcrumb("B", nil))
	bcSmall.Push(NewBreadcrumb("C", nil))
	bcSmall.SetRect(0, 0, 1, 1) // width = 1, InnerRect width will be 1
	// bcSmall.Draw(s) // Avoid Draw if it resets state
	handlerSmall := bcSmall.MouseHandler()
	// Mouse event at 0,0 is inside 1x1 rect.
	// In the loop:
	// i=0: item A. cursorX=0, maxX=1. label="A", w=1.
	//      x=0, cursorX=0, w=1 -> x >= cursorX && x < cursorX+w is true.
	//      It will return true, nil.
	// We need it to NOT enter the if, but continue the loop and hit cursorX >= maxX.
	// So we need x to be out of the first item but still in InnerRect.
	// If width=2, InnerRect width=2.
	// i=0: A. cursorX=0, maxX=2. label="A", w=1.
	//      If x=1, it's not in A.
	//      cursorX becomes 1.
	//      i=0 < len-1(2) and cursorX(1) < maxX(2) -> separator " > ". sepW=3.
	//      cursorX becomes 4.
	// i=1: B. cursorX=4 >= maxX=2 -> BREAK!
	bcSmall.SetRect(0, 0, 2, 1)
	ev = tcell.NewEventMouse(1, 0, tcell.Button1, 0)
	_, _ = handlerSmall(tview.MouseLeftClick, ev, func(p tview.Primitive) {})
}

func TestBreadcrumbs_InputHandler_EdgeCases(t *testing.T) {
	t.Parallel()
	// Empty items
	bc := NewBreadcrumbs(nil)
	bc.items = nil // Force it to be empty as NewBreadcrumbs(nil) might add a home item or something
	handler := bc.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil) // should return early

	// Focus transition without next target
	bc2 := NewBreadcrumbs(NewBreadcrumb("A", nil))
	bc2.SetRect(0, 0, 20, 1)
	handler2 := bc2.InputHandler()
	var focused tview.Primitive = bc2
	handler2(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), func(p tview.Primitive) { focused = p })
	if focused != nil {
		t.Errorf("Expected focus to be nil, got %v", focused)
	}

	// Default case (some other key)
	handler2(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone), nil)
}

func TestBreadcrumbs_FocusBlur_EdgeCases(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("A", nil))
	bc.Push(NewBreadcrumb("B", nil))

	// Focus when selection is valid (should keep it)
	bc.selectedItemIndex = 0
	bc.Focus(func(p tview.Primitive) {})
	if bc.selectedItemIndex != 0 {
		t.Errorf("Expected selection to remain 0, got %d", bc.selectedItemIndex)
	}

	// Focus when selection is invalid
	bc.selectedItemIndex = 5
	bc.Focus(func(p tview.Primitive) {})
	if bc.selectedItemIndex != 0 { // len(items)-2 = 2-2 = 0
		t.Errorf("Expected selection to be 0, got %d", bc.selectedItemIndex)
	}

	bc.Blur()
	if bc.selectedItemIndex != 1 { // len(items)-1 = 1
		t.Errorf("Expected selection to be 1 after Blur, got %d", bc.selectedItemIndex)
	}
}
