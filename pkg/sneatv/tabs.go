package sneatv

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TabStyles struct {
	Foreground string
	Background string
}

type TabsStyle struct {
	Radio      bool
	Underscore bool

	ActiveFocused   TabStyles
	ActiveBlur      TabStyles
	InactiveFocused TabStyles
	InactiveBlur    TabStyles
}

// Tab represents a single tab.
type Tab struct {
	ID       string
	Title    string
	Closable bool
	tview.Primitive
}

func NewTab(id string, title string, closable bool, content tview.Primitive) *Tab {
	return &Tab{
		ID:        id,
		Title:     title,
		Closable:  closable,
		Primitive: content,
	}
}

type tabsApp interface {
	QueueUpdateDraw(f func()) *tview.Application
	SetFocus(p tview.Primitive) *tview.Application
}

// Tabs is a tab container implemented using tview.Pages.
type Tabs struct {
	*tview.Flex
	tabsOptions
	TabsStyle

	app tabsApp

	TextView *tview.TextView // TODO(help-wanted): exported as a workaround to set focus - needs fix!
	pages    *tview.Pages

	isFocused bool

	tabs   []*Tab
	active int

	focusFunc               func()
	textViewFocusFunc       func()
	textViewBlurFunc        func()
	textViewHighlightedFunc func(added, removed, remaining []string)
}

type tabsOptions struct {
	label      string
	focusDown  func(current tview.Primitive)
	focusLeft  func(current tview.Primitive)
	focusRight func(current tview.Primitive)
	focusUp    func(current tview.Primitive)
}

type TabsOption func(*tabsOptions)

func WithLabel(label string) TabsOption {
	return func(o *tabsOptions) {
		o.label = label
	}
}

func FocusDown(f func(current tview.Primitive)) TabsOption {
	return func(o *tabsOptions) {
		o.focusDown = f
	}
}

func FocusRight(f func(current tview.Primitive)) TabsOption {
	return func(o *tabsOptions) {
		o.focusRight = f
	}
}

func FocusUp(f func(current tview.Primitive)) TabsOption {
	return func(o *tabsOptions) {
		o.focusUp = f
	}
}

func FocusLeft(f func(current tview.Primitive)) TabsOption {
	return func(o *tabsOptions) {
		o.focusLeft = f
	}
}

var UnderlineTabsStyle = TabsStyle{
	Radio:      false,
	Underscore: true,
	ActiveFocused: TabStyles{
		Foreground: "black",
		Background: "lightgray",
	},
	ActiveBlur: TabStyles{
		Foreground: "black",
		Background: "darkgray",
	},
	InactiveFocused: TabStyles{
		Foreground: "lightgray",
		Background: "black",
	},
	InactiveBlur: TabStyles{
		Foreground: "gray",
		Background: "black",
	},
}

var RadioTabsStyle = TabsStyle{
	Radio:      true,
	Underscore: false,
	ActiveFocused: TabStyles{
		Foreground: "white",
		Background: "black",
	},
	ActiveBlur: TabStyles{
		Foreground: "lightgray",
		Background: "black",
	},
	InactiveFocused: TabStyles{
		Foreground: "lightgray",
		Background: "black",
	},
}

//func (t *Tabs) TakeFocus() {
//	t.app.SetFocus(t.TextView)
//}

// NewTabs creates a new tab container.
func NewTabs(app tabsApp, style TabsStyle, options ...TabsOption) *Tabs {
	pages := tview.NewPages()

	t := &Tabs{
		app:       app,
		active:    -1,
		TabsStyle: style,
		pages:     pages,
		Flex:      tview.NewFlex().SetDirection(tview.FlexRow),
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWrap(false),
	}
	t.focusFunc = func() {
		if t.app != nil {
			t.app.SetFocus(t.TextView)
		}
	}
	t.SetFocusFunc(t.focusFunc)
	for _, set := range options {
		set(&t.tabsOptions)
	}

	t.TextView.SetInputCapture(t.handleInput)

	t.textViewFocusFunc = func() {
		t.setIsFocused(true)
	}
	t.TextView.SetFocusFunc(t.textViewFocusFunc)

	t.textViewBlurFunc = func() {
		t.setIsFocused(false)
	}
	t.TextView.SetBlurFunc(t.textViewBlurFunc)

	t.textViewHighlightedFunc = func(added, removed, remaining []string) {
		if len(added) == 0 {
			return
		}

		region := added[0]

		var index int
		if _, err := fmt.Sscanf(region, "tab-%d", &index); err != nil {
			return
		}
		//t.tabs[index].Title = fmt.Sprintf("Tab %d", index)
		t.SwitchTo(index)
	}
	t.TextView.SetHighlightedFunc(t.textViewHighlightedFunc)

	t.
		AddItem(t.TextView, 1, 0, false).
		AddItem(pages, 0, 1, true)

	return t
}

func (t *Tabs) setIsFocused(isFocused bool) {
	t.isFocused = isFocused
	if t.app != nil {
		go t.app.QueueUpdateDraw(func() {
			t.updateTextView()
		})
	} else {
		t.updateTextView()
	}
}

// AddTabs adds new tabs.
func (t *Tabs) AddTabs(tabs ...*Tab) {
	is1stTab := len(t.tabs) == 0
	t.tabs = append(t.tabs, tabs...)

	for _, tab := range tabs {
		t.pages.AddPage(
			tab.ID,
			tab.Primitive,
			true,
			is1stTab,
		)
	}

	if is1stTab {
		t.SwitchTo(0)
	}
}

// SwitchTo switches to a tab by index.
func (t *Tabs) SwitchTo(index int) {
	if index < 0 || index >= len(t.tabs) {
		return
	}
	if t.active == index {
		return
	}
	t.active = index
	t.pages.SwitchToPage(t.tabs[index].ID)
	t.updateTextView()
	t.TextView.Highlight()
	//tab := t.tabs[index]
	//i := strconv.Itoa(index)
	//if tab.Closable {
	//
	//	t.TextView.Highlight("tab-"+i, "close-"+i, "l-"+i)
	//} else {
	//	t.TextView.Highlight("tab-" + i)
	//}
}

// updateTextView redraws the tab bar.
func (t *Tabs) updateTextView() {
	t.TextView.Clear()

	if t.label != "" {
		_, _ = t.TextView.Write([]byte(t.label))
	}

	const bold = "b"
	const underline = "u"

	for i, tab := range t.tabs {
		isActive := i == t.active
		var title string
		if t.Radio {
			if i == t.active {
				title = "◉ " + tab.Title
			} else {
				title = "○ " + tab.Title
			}
		} else {
			title = tab.Title
		}

		region := fmt.Sprintf("tab-%d", i)

		var fontStyle string
		var fg string
		var bg string

		if isActive {
			if t.isFocused {
				fontStyle = bold
				fg = t.ActiveFocused.Foreground
				bg = t.ActiveFocused.Background
			} else {
				fg = t.ActiveBlur.Foreground
				bg = t.ActiveBlur.Background
			}
		} else {
			if t.isFocused {
				fg = t.InactiveFocused.Foreground
				bg = t.InactiveFocused.Background
			} else {
				fg = t.InactiveBlur.Foreground
				bg = t.InactiveBlur.Background
			}
			if t.Underscore {
				fontStyle = underline
			}
		}
		if fontStyle == "" {
			_, _ = fmt.Fprintf(t.TextView, `["%s"][%s:%s] %s [-:-][""]`, region, fg, bg, title)
		} else {
			_, _ = fmt.Fprintf(t.TextView, `["%s"][%s:%s:%s] %s [-:-:%s][""]`,
				region, fg, bg, fontStyle, title, strings.ToUpper(fontStyle))
		}
		if tab.Closable {
			if t.Underscore {
				if isActive {
					_, _ = fmt.Fprintf(t.TextView, `["close-%d"][%s:%s]✖ [-:-][""][::u] [::U]`, i, fg, bg)
				} else {
					_, _ = fmt.Fprintf(t.TextView, `["close-%d"][%s:%s:u]✖  [-:-:U][""]`, i, fg, bg)
				}
			} else {
				_, _ = fmt.Fprintf(t.TextView, `["close-%d"][%s:%s]✖ [-:-][""] `, i, fg, bg)
			}
		}
	}
}

// handleInput handles keyboard navigation.
func (t *Tabs) handleInput(ev *tcell.EventKey) *tcell.EventKey {
	switch ev.Key() {
	case tcell.KeyRight:
		if t.active == len(t.tabs)-1 {
			if t.focusRight != nil {
				t.focusRight(t.TextView)
			}
			return nil
		}
		t.SwitchTo((t.active + 1) % len(t.tabs))
		return nil
	case tcell.KeyLeft:
		if t.active == 0 {
			if t.focusLeft != nil {
				t.focusLeft(t.TextView)
			}
			return nil
		}
		t.SwitchTo((t.active - 1 + len(t.tabs)) % len(t.tabs))
		return nil
	case tcell.KeyUp:
		if t.focusUp != nil {
			t.focusUp(t.TextView)
		}
		return nil
	case tcell.KeyDown:
		if t.focusDown != nil {
			t.focusDown(t.TextView)
		}
		return nil
	default:
		if ev.Modifiers() == tcell.ModAlt {
			if ev.Rune() >= '1' && ev.Rune() <= '9' {
				t.SwitchTo(int(ev.Rune() - '1'))
				return nil
			}
		}
		return ev
	}
}
