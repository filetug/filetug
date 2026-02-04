package filetug

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewContainer(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
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
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
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

func TestContainer_SetContent_NilSafety(t *testing.T) {
	t.Parallel()
	var nilContainer *Container
	assert.NotPanics(t, func() {
		nilContainer.SetContent(nil)
	})

	empty := &Container{}
	assert.NotPanics(t, func() {
		empty.SetContent(tview.NewBox())
	})
}

func TestContainer_Focus(t *testing.T) {
	t.Parallel()
	t.Run("non_nil_content", func(t *testing.T) {
		b := tview.NewBox()

		nav, app, _ := newNavigatorForTest(t)
		app.EXPECT().SetFocus(b).AnyTimes()

		c := NewContainer(1, nav)
		c.SetContent(b)

		c.Focus(func(p tview.Primitive) {
			assert.Equal(t, b, p)
		})
	})

	t.Run("nil_content", func(t *testing.T) {
		nav, app, _ := newNavigatorForTest(t)
		c := NewContainer(1, nav)
		app.EXPECT().SetFocus(c).AnyTimes()

		c.Focus(func(p tview.Primitive) {
			assert.Equal(t, c, p)
		})
	})
}
