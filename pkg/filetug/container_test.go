package filetug

import (
	"testing"

	"github.com/rivo/tview"
)

func TestNewContainer(t *testing.T) {
	app := tview.NewApplication()
	nav := &Navigator{app: app}
	index := 1
	c := newContainer(index, nav)

	if c == nil {
		t.Fatal("Expected container to be not nil")
	}
	if c.index != index {
		t.Errorf("Expected index %d, got %d", index, c.index)
	}
	if c.nav != nav {
		t.Errorf("Expected nav to be set")
	}
	if c.Flex == nil {
		t.Error("Expected Flex to be initialized")
	}
}

func TestContainer_SetContent(t *testing.T) {
	app := tview.NewApplication()
	nav := &Navigator{app: app}
	c := newContainer(1, nav)

	p := tview.NewBox()
	c.SetContent(p)

	if c.inner != p {
		t.Error("Expected inner to be set to p")
	}

	// Verify AddItem was called (indirectly by checking children count)
	if c.GetItemCount() != 1 {
		t.Errorf("Expected 1 item in Flex, got %d", c.GetItemCount())
	}

	// Verify Clear and AddItem work by setting content again
	p2 := tview.NewBox()
	c.SetContent(p2)
	if c.inner != p2 {
		t.Error("Expected inner to be updated to p2")
	}
	if c.GetItemCount() != 1 {
		t.Errorf("Expected 1 item in Flex after second SetContent, got %d", c.GetItemCount())
	}
}

func TestContainer_Focus(t *testing.T) {
	app := tview.NewApplication()
	nav := &Navigator{
		app: app,
		setAppFocus: func(p tview.Primitive) {
			app.SetFocus(p)
		},
	}
	c := newContainer(1, nav)

	p := tview.NewBox()
	c.SetContent(p)

	// Test case when inner is not nil
	c.Focus(func(p tview.Primitive) {})
	if app.GetFocus() != p {
		t.Errorf("Expected focus to be set to inner primitive, got %v", app.GetFocus())
	}

	// Test case when inner is nil
	app.SetFocus(nil)
	c.inner = nil
	c.Focus(func(p tview.Primitive) {})
	if app.GetFocus() != nil {
		t.Errorf("Expected focus to remain nil when inner is nil, got %v", app.GetFocus())
	}
}
