package filetug

import (
	"testing"

	"github.com/rivo/tview"
)

func TestNewContainer(t *testing.T) {
	nav := NewNavigator(nil)
	index := 1
	c := NewContainer(index, nav)

	if c == nil {
		t.Fatal("Expected Container to be not nil")
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
	nav := NewNavigator(nil)
	c := NewContainer(1, nav)

	p := tview.NewBox()
	c.SetContent(p)

	if c.content != p {
		t.Error("Expected content to be set to p")
	}

	// Verify AddItem was called (indirectly by checking children count)
	if c.GetItemCount() != 1 {
		t.Errorf("Expected 1 item in Flex, got %d", c.GetItemCount())
	}

	// Verify Clear and AddItem work by setting content again
	p2 := tview.NewBox()
	c.SetContent(p2)
	if c.content != p2 {
		t.Error("Expected content to be updated to p2")
	}
	if c.GetItemCount() != 1 {
		t.Errorf("Expected 1 item in Flex after second SetContent, got %d", c.GetItemCount())
	}
}

func TestContainer_Focus(t *testing.T) {
	app := tview.NewApplication()
	nav := &Navigator{
		setAppFocus: func(p tview.Primitive) {
			app.SetFocus(p)
		},
	}
	c := NewContainer(1, nav)

	p := tview.NewBox()
	c.SetContent(p)

	// Test case when content is not nil
	c.Focus(func(p tview.Primitive) {})
	if app.GetFocus() != p {
		t.Errorf("Expected focus to be set to content primitive, got %v", app.GetFocus())
	}

	// Test case when content is nil
	app.SetFocus(nil)
	c.content = nil
	c.Focus(func(p tview.Primitive) {})
	if app.GetFocus() != nil {
		t.Errorf("Expected focus to remain nil when content is nil, got %v", app.GetFocus())
	}
}
