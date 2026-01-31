package filetug

import (
	"fmt"
	"testing"

	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/stretchr/testify/assert"
)

func Test_bottom_getCtrlMenuItems(t *testing.T) {
	t.Parallel()
	b := &bottom{}
	menuItems := b.getCtrlMenuItems()
	assert.Len(t, menuItems, 4)
}

func TestNewBottom(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)
	assert.NotNil(t, b)
	assert.NotNil(t, b.TextView)
	assert.NotEmpty(t, b.altMenuItems)
}

func TestBottom_Render(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)
	b.render()
	text := b.GetText(false)
	// It's [white]Alt[-]+:
	assert.Contains(t, text, "Alt")
	assert.Contains(t, text, "x") // for Exit (it's E[white]x[-]it)
}

func TestBottom_Highlighted(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)

	actionCalled := false
	b.altMenuItems = []ftui.MenuItem{
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
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)
	items := b.getAltMenuItems()
	assert.NotEmpty(t, items)

	// Ensure we cover all items in getAltMenuItems
	for i := range items {
		_ = items[i].Title
	}

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

func TestBottom_GetAltMenuItems_Actions(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)
	items := b.getAltMenuItems()

	for _, item := range items {
		// Avoid Exit as it calls os.Exit(0)
		// Also avoid anything that might hang if app is not running.
		if item.Title != "Exit" && item.Action != nil {
			// To be safe, we could check if action interacts with app.
			// But since we just want coverage, and most are empty funcs currently:
			item.Action()
		}
	}
}

func TestBottom_GetAltMenuItems_ExitAction(t *testing.T) {
	t.Parallel()
	nav, app, _ := newNavigatorForTest(t)
	b := newBottom(nav)

	app.EXPECT().Stop().Times(1)

	exitCode := -1
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	osExit = func(code int) { exitCode = code }

	var exitAction func()
	for _, item := range b.getAltMenuItems() {
		if item.Title == "Exit" {
			exitAction = item.Action
			break
		}
	}
	if exitAction == nil {
		t.Fatal("exit action is nil")
	}

	exitAction()

	assert.Equal(t, 0, exitCode)
}

func TestBottom_RenderMenuItems_ReplacesHotkeysAndRegions(t *testing.T) {
	t.Parallel()
	b := &bottom{}
	menuItems := []ftui.MenuItem{
		{Title: "/root", HotKeys: []string{"/"}},
		{Title: "~Home", HotKeys: []string{"~"}},
		{Title: "F6Rename", HotKeys: []string{"F6", "R"}},
	}

	text := b.renderMenuItems(menuItems)
	assert.Contains(t, text, `["root"]`)
	assert.Contains(t, text, `["home"]`)

	hotkeyF6 := fmt.Sprintf("[%s]%s[-]", sneatv.CurrentTheme.HotkeyColor, "F6")
	hotkeyR := fmt.Sprintf("[%s]%s[-]", sneatv.CurrentTheme.HotkeyColor, "R")
	assert.Contains(t, text, hotkeyF6)
	assert.Contains(t, text, hotkeyR)
}

func TestBottom_Highlighted_Ctrl(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	b := newBottom(nav)
	b.isCtrl = true

	archiveAction()

	actionCalled := false
	oldArchiveAction := archiveAction
	defer func() { archiveAction = oldArchiveAction }()
	archiveAction = func() {
		actionCalled = true
	}
	// We need to set a mock item in CTRL menu if we wanted to test action,
	// but getCtrlMenuItems is hardcoded and doesn't have actions.
	// So we just call it to cover the branches.
	b.highlighted([]string{"A"}, nil, nil)
	assert.True(t, actionCalled)

	// Test no action for unknown region in CTRL menu
	actionCalled = false
	b.highlighted([]string{"Z"}, nil, nil)
	assert.False(t, actionCalled)
}
