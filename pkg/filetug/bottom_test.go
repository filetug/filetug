package filetug

import (
	"testing"

	"github.com/datatug/filetug/pkg/filetug/ftui"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func Test_bottom_getCtrlMenuItems(t *testing.T) {
	b := &bottom{}
	menuItems := b.getCtrlMenuItems()
	assert.Len(t, menuItems, 4)
}

func TestNewBottom(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	b := newBottom(nav)
	assert.NotNil(t, b)
	assert.NotNil(t, b.TextView)
	assert.NotEmpty(t, b.menuItems)
}

func TestBottom_Render(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	b := newBottom(nav)
	b.render()
	text := b.GetText(false)
	// It's [white]Alt[-]+:
	assert.Contains(t, text, "Alt")
	assert.Contains(t, text, "x") // for Exit (it's E[white]x[-]it)
}

func TestBottom_Highlighted(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	b := newBottom(nav)

	actionCalled := false
	b.menuItems = []ftui.MenuItem{
		{
			Title:   "Test",
			HotKeys: []string{"T"},
			Action:  func() { actionCalled = true },
		},
	}

	// Test action call
	b.highlighted([]string{"T"}, nil, nil)
	assert.True(t, actionCalled)

	// Test no action for unknown region
	actionCalled = false
	b.highlighted([]string{"Unknown"}, nil, nil)
	assert.False(t, actionCalled)

	// Test empty added
	b.highlighted([]string{}, nil, nil)
	assert.False(t, actionCalled)
}

func TestBottom_GetAltMenuItems(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	b := newBottom(nav)
	items := b.getAltMenuItems()
	assert.NotEmpty(t, items)

	// Find Exit item and check it doesn't crash (mocking os.Exit is hard, but we can check it exists)
	var exitItem *ftui.MenuItem
	for _, item := range items {
		if item.Title == "Exit" {
			exitItem = &item
			break
		}
	}
	if exitItem == nil {
		t.Fatal("exitItem is nil")
	}
	assert.NotNil(t, exitItem.Action)
}
