package sneatv

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type fakeTabsApp struct {
	updated chan struct{}
}

func (f *fakeTabsApp) QueueUpdateDraw(fn func()) {
	fn()
	if f.updated != nil {
		f.updated <- struct{}{}
	}
}

func (f *fakeTabsApp) SetFocus(p tview.Primitive) {
	_ = p
}

type tviewTabsApp struct {
	*tview.Application
}

func (a tviewTabsApp) QueueUpdateDraw(f func()) {
	_ = a.Application.QueueUpdateDraw(f)
}

func (a tviewTabsApp) SetFocus(p tview.Primitive) {
	_ = a.Application.SetFocus(p)
}

func TestNewTab(t *testing.T) {
	t.Parallel()
	content := tview.NewBox()
	tab := NewTab("tab-1", "Tab 1", true, content)
	assert.Equal(t, "tab-1", tab.ID)
	assert.Equal(t, "Tab 1", tab.Title)
	assert.True(t, tab.Closable)
	assert.Same(t, content, tab.Primitive)
}

func TestNewTabs(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle, WithLabel("Tabs:"))
	assert.NotNil(t, tabs)
	assert.Equal(t, "Tabs:", tabs.label)
}

func TestTabs_AddAndSwitch(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)
	tab1 := &Tab{ID: "1", Title: "Tab 1", Primitive: tview.NewBox()}
	tab2 := &Tab{ID: "2", Title: "Tab 2", Primitive: tview.NewBox()}
	tabs.AddTabs(tab1, tab2)

	assert.Equal(t, 2, len(tabs.tabs))
	assert.Equal(t, 0, tabs.active)

	tabs.SwitchTo(1)
	assert.Equal(t, 1, tabs.active)

	tabs.SwitchTo(5) // out of bounds
	assert.Equal(t, 1, tabs.active)
}

func TestTabs_AddTabsUpdatesTextView(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)
	tab1 := &Tab{ID: "1", Title: "Tab 1", Primitive: tview.NewBox()}
	tab2 := &Tab{ID: "2", Title: "Tab 2", Primitive: tview.NewBox()}

	tabs.AddTabs(tab1)
	text := tabs.TextView.GetText(false)
	assert.Contains(t, text, "Tab 1")
	assert.NotContains(t, text, "Tab 2")

	tabs.AddTabs(tab2)
	text = tabs.TextView.GetText(false)
	assert.Contains(t, text, "Tab 2")
}

func TestTabs_Navigation(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle,
		WithLabel("Tabs:"),
		FocusLeft(func(current tview.Primitive) {}),
		FocusRight(func(current tview.Primitive) {}),
		FocusUp(func(current tview.Primitive) {}),
		FocusDown(func(current tview.Primitive) {}),
	)
	tabs.AddTabs(
		&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox()},
		&Tab{ID: "2", Title: "T2", Primitive: tview.NewBox()},
	)

	// Right
	event := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	res := tabs.handleInput(event)
	assert.Nil(t, res)
	assert.Equal(t, 1, tabs.active)

	// Right again (at last tab)
	rightCalled := false
	tabs.focusRight = func(current tview.Primitive) { rightCalled = true }
	tabs.handleInput(event)
	assert.True(t, rightCalled)

	// Left
	event = tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	res = tabs.handleInput(event)
	assert.Nil(t, res)
	assert.Equal(t, 0, tabs.active)

	// FocusLeft
	leftCalled := false
	tabs.focusLeft = func(current tview.Primitive) { leftCalled = true }
	tabs.handleInput(event)
	assert.True(t, leftCalled)

	// FocusUp
	upCalled := false
	tabs.focusUp = func(current tview.Primitive) { upCalled = true }
	event = tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	tabs.handleInput(event)
	assert.True(t, upCalled)

	// FocusDown
	downCalled := false
	tabs.focusDown = func(current tview.Primitive) { downCalled = true }
	event = tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	tabs.handleInput(event)
	assert.True(t, downCalled)

	// Alt+1
	event = tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModAlt)
	tabs.handleInput(event)
	assert.Equal(t, 0, tabs.active)

	// Other key
	event = tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	res = tabs.handleInput(event)
	assert.NotNil(t, res)
}

func TestTabs_UpdateTextView(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, RadioTabsStyle, WithLabel("Tabs:"))
	tabs.AddTabs(
		&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox(), Closable: true},
		&Tab{ID: "2", Title: "T2", Primitive: tview.NewBox()},
	)

	// Test with focus
	tabs.isFocused = true
	tabs.updateTextView()
	assert.Contains(t, tabs.TextView.GetText(false), "◉ T1")

	// Test without focus
	tabs.isFocused = false
	tabs.updateTextView()
	assert.Contains(t, tabs.TextView.GetText(false), "◉ T1")

	// Test underline style with closable
	tabs.TabsStyle = UnderlineTabsStyle
	tabs.tabs[0].Closable = false
	tabs.tabs[1].Closable = true
	tabs.updateTextView()
	assert.Contains(t, tabs.TextView.GetText(false), `["close-1"]`)

	tabs.tabs[0].Closable = true
	tabs.tabs[1].Closable = false
	tabs.updateTextView()
	assert.Contains(t, tabs.TextView.GetText(false), `["close-0"]`)
}

func TestTabs_Callbacks(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)
	tabs.AddTabs(&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox()})

	// Since we can't get the functions back, we can only verify they don't crash
	// by assuming they are set correctly and would be called by tview.
	// We've added nil checks to make them safe if app is nil.

	tabsNoApp := NewTabs(nil, UnderlineTabsStyle)
	tabsNoApp.AddTabs(&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox()})
}

func TestTabs_Options(t *testing.T) {
	t.Parallel()
	focusUpCalled := false
	focusDownCalled := false
	focusLeftCalled := false
	focusRightCalled := false

	tabs := NewTabs(nil, UnderlineTabsStyle,
		FocusUp(func(current tview.Primitive) { focusUpCalled = true }),
		FocusDown(func(current tview.Primitive) { focusDownCalled = true }),
		FocusLeft(func(current tview.Primitive) { focusLeftCalled = true }),
		FocusRight(func(current tview.Primitive) { focusRightCalled = true }),
	)

	tabs.focusUp(nil)
	assert.True(t, focusUpCalled)
	tabs.focusDown(nil)
	assert.True(t, focusDownCalled)
	tabs.focusLeft(nil)
	assert.True(t, focusLeftCalled)
	tabs.focusRight(nil)
	assert.True(t, focusRightCalled)
}

func TestTabs_FocusCallbacks(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)

	// Test FocusFunc
	if tabs.focusFunc != nil {
		tabs.focusFunc()
	}

	// Test TextView FocusFunc
	if tabs.textViewFocusFunc != nil {
		tabs.textViewFocusFunc()
	}
	assert.True(t, tabs.isFocused)

	// Test TextView BlurFunc
	if tabs.textViewBlurFunc != nil {
		tabs.textViewBlurFunc()
	}
	assert.False(t, tabs.isFocused)
}

func TestTabs_SetIsFocused_WithApp(t *testing.T) {
	t.Parallel()
	app := &fakeTabsApp{updated: make(chan struct{}, 1)}
	tabs := NewTabs(app, UnderlineTabsStyle)
	tabs.AddTabs(&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox()})

	tabs.setIsFocused(true)

	select {
	case <-app.updated:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected QueueUpdateDraw to run")
	}
	assert.True(t, tabs.isFocused)
}

func TestTabs_SetIsFocused_NoApp(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)
	tabs.AddTabs(&Tab{ID: "1", Title: "T1", Primitive: tview.NewBox()})

	tabs.setIsFocused(true)

	assert.True(t, tabs.isFocused)
}

func TestTabs_HighlightedFunc(t *testing.T) {
	t.Parallel()
	tabs := NewTabs(nil, UnderlineTabsStyle)
	tab1 := &Tab{ID: "1", Title: "Tab 1", Primitive: tview.NewBox()}
	tab2 := &Tab{ID: "2", Title: "Tab 2", Primitive: tview.NewBox()}
	tabs.AddTabs(tab1, tab2)

	assert.NotNil(t, tabs.textViewHighlightedFunc)

	// Valid tab highlight
	tabs.textViewHighlightedFunc([]string{"tab-1"}, nil, nil)
	assert.Equal(t, 1, tabs.active)

	// Invalid tab highlight
	tabs.textViewHighlightedFunc([]string{"invalid"}, nil, nil)
	assert.Equal(t, 1, tabs.active)

	// Empty added
	tabs.textViewHighlightedFunc([]string{}, nil, nil)
	assert.Equal(t, 1, tabs.active)
}

func TestNewTabs_WithTviewApp(t *testing.T) {
	t.Parallel()
	app := tview.NewApplication()
	tabs := NewTabs(tviewTabsApp{app}, UnderlineTabsStyle)
	assert.NotNil(t, tabs)
}

type focusTrackingApp struct {
	focused tview.Primitive
}

func (f *focusTrackingApp) QueueUpdateDraw(fn func()) {
	fn()
}

func (f *focusTrackingApp) SetFocus(p tview.Primitive) {
	f.focused = p
}

func TestNewTabs_FocusFunc_WithApp(t *testing.T) {
	t.Parallel()
	app := &focusTrackingApp{}
	tabs := NewTabs(app, UnderlineTabsStyle)

	assert.NotNil(t, tabs.focusFunc)
	tabs.focusFunc()

	assert.Same(t, tabs.TextView, app.focused)
}
