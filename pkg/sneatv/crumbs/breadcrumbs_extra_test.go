package crumbs

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBreadcrumbs_GoHome(t *testing.T) {
	t.Parallel()
	homeCalled := false
	home := NewBreadcrumb("Home", func() error {
		homeCalled = true
		return nil
	})
	bc := NewBreadcrumbs(home)
	bc.Push(NewBreadcrumb("Child", nil))

	err := bc.GoHome()
	if err != nil {
		t.Errorf("GoHome returned error: %v", err)
	}
	if !homeCalled {
		t.Errorf("GoHome did not call the home action")
	}
}

func TestBreadcrumbs_TakeFocus(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	bc.Push(NewBreadcrumb("Child", nil))
	bc.selectedItemIndex = 1

	target := tview.NewBox()
	bc.TakeFocus(target)

	if bc.selectedItemIndex != 0 {
		t.Errorf("expected selectedItemIndex to be 0, got %d", bc.selectedItemIndex)
	}
	if bc.nextFocusTarget != target {
		t.Errorf("expected nextFocusTarget to be set")
	}
}

func TestBreadcrumbs_IsLastItemSelected(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))

	bc.Push(NewBreadcrumb("Child", nil))
	bc.selectedItemIndex = 1
	if !bc.IsLastItemSelected() {
		t.Errorf("expected IsLastItemSelected to be true")
	}

	bc.selectedItemIndex = 0
	if bc.IsLastItemSelected() {
		t.Errorf("expected IsLastItemSelected to be false")
	}
}

func TestBreadcrumbs_FocusBlur(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	bc.Push(NewBreadcrumb("Child", nil))

	// Test Blur
	bc.Blur()
	if bc.selectedItemIndex != 1 {
		t.Errorf("Blur should select last item, got %d", bc.selectedItemIndex)
	}

	// Test Focus
	bc.Focus(func(p tview.Primitive) {})
	// items: Home(0), Child(1).
	// len=2. selectedItemIndex=1.
	// b.selectedItemIndex < 0 || b.selectedItemIndex >= len(b.items)-1 is false because 1 < 0 is false and 1 >= 1 is true.
	// Wait, b.selectedItemIndex >= len(b.items)-1. 1 >= 2-1 is true.
	// So it should set b.selectedItemIndex = 2 - 2 = 0.
	if bc.selectedItemIndex != 0 {
		t.Errorf("Focus should select second to last item if last was selected, got %d", bc.selectedItemIndex)
	}
}

func TestBreadcrumbs_FocusTargets(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	next := tview.NewBox()
	prev := tview.NewBox()

	bc.SetNextFocusTarget(next)
	bc.SetPrevFocusTarget(prev)

	if bc.nextFocusTarget != next {
		t.Errorf("SetNextFocusTarget failed")
	}
	if bc.prevFocusTarget != prev {
		t.Errorf("SetPrevFocusTarget failed")
	}
}

func TestBreadcrumbs_Options(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil), WithSeparatorStartIndex(1))
	if bc.separatorStartIdx != 1 {
		t.Errorf("WithSeparatorStartIndex failed, got %d", bc.separatorStartIdx)
	}
}

func TestBreadcrumbs_Draw_Extras(t *testing.T) {
	t.Parallel()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 5)

	bc := NewBreadcrumbs(NewBreadcrumb("http://example.com", nil))
	bc.Push(NewBreadcrumb("Colored", nil).SetColor(tcell.ColorRed))
	bc.SetRect(0, 0, 80, 5)

	// Test drawing with focus
	bc.Focus(func(p tview.Primitive) {})
	bc.Draw(s)

	// Test drawing with width 0
	bc.SetRect(0, 0, 0, 5)
	bc.Draw(s)
}

func TestBreadcrumbs_InputHandler_TabDown(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	next := tview.NewBox()
	bc.SetNextFocusTarget(next)

	var focused tview.Primitive
	setFocus := func(p tview.Primitive) {
		focused = p
	}

	handler := bc.InputHandler()

	// Test Tab
	handler(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), setFocus)
	if focused != next {
		t.Errorf("Tab should have focused next target")
	}

	// Test Down
	focused = nil
	handler(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), setFocus)
	if focused != next {
		t.Errorf("Down should have focused next target")
	}

	// Test Tab without target
	bc.SetNextFocusTarget(nil)
	focused = next // just to see it changes
	handler(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), setFocus)
	if focused != nil {
		t.Errorf("Tab without target should have focused nil")
	}
}

func TestBreadcrumbs_InputHandler_OtherKeys(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	bc.Push(NewBreadcrumb("Child", nil))
	bc.Push(NewBreadcrumb("Grandchild", nil))

	setFocus := func(p tview.Primitive) {}
	handler := bc.InputHandler()

	// Test Left
	bc.selectedItemIndex = 2
	handler(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), setFocus)
	if bc.selectedItemIndex != 1 {
		t.Errorf("Left key failed, got %d", bc.selectedItemIndex)
	}

	// Test Right
	handler(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), setFocus)
	if bc.selectedItemIndex != 2 {
		t.Errorf("Right key failed, got %d", bc.selectedItemIndex)
	}

	// Test Right at end
	handler(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), setFocus)
	if bc.selectedItemIndex != 2 {
		t.Errorf("Right key at end should stay at end, got %d", bc.selectedItemIndex)
	}

	// Test Enter
	actionCalled := false
	bc.items[2] = NewBreadcrumb("Grandchild", func() error {
		actionCalled = true
		return nil
	})
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), setFocus)
	if !actionCalled {
		t.Errorf("Enter key did not call action")
	}

	// Test default key (not handled)
	handler(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone), setFocus)
}

func TestBreadcrumbs_InputHandler_Empty(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	bc.items = nil
	handler := bc.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
}

func TestBreadcrumbs_MouseHandler_Focus(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumbs(NewBreadcrumb("Home", nil))
	bc.Push(NewBreadcrumb("Child", nil))
	bc.SetRect(0, 0, 80, 5)

	handler := bc.MouseHandler()

	var focused tview.Primitive
	setFocus := func(p tview.Primitive) {
		focused = p
	}

	// Click inside but after items (on the right)
	ev := tcell.NewEventMouse(70, 0, tcell.Button1, 0)
	consumed, _ := handler(tview.MouseLeftClick, ev, setFocus)
	if !consumed {
		t.Errorf("Click on the right should be consumed")
	}
	if focused != bc {
		t.Errorf("Click on the right should focus bc")
	}
}
