package filetug

import (
	"fmt"
	"os"
	"strings"

	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type bottom struct {
	*tview.TextView
	nav          *Navigator
	altMenuItems []ftui.MenuItem
	fkMenuItems  []ftui.MenuItem
	isCtrl       bool
}

func newBottom(nav *Navigator) *bottom {
	b := &bottom{
		nav: nav,
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetTextColor(tcell.ColorSlateGray),
	}

	b.SetHighlightedFunc(b.highlighted)

	b.altMenuItems = b.getAltMenuItems()
	b.fkMenuItems = b.getFKMenuItems()

	b.render()

	return b
}

func (b *bottom) render() {

	var sb strings.Builder

	{
		menuItemsText := b.renderMenuItems(b.fkMenuItems)
		sb.WriteString(menuItemsText)
	}
	sb.WriteString("  ║  ")
	{
		sb.WriteString("[blue]Alt+[-]")
		menuItemsText := b.renderMenuItems(b.altMenuItems)
		sb.WriteString(menuItemsText)
	}

	text := sb.String()
	b.SetText(text)
}

func (b *bottom) renderMenuItems(menuItems []ftui.MenuItem) string {
	const separator = "┊"
	var sb strings.Builder
	for _, mi := range menuItems {
		title := mi.Title
		for _, key := range mi.HotKeys {
			color := "white"
			if mi.IsAltHotkey {
				color = "blue"
			}
			hotkeyText := fmt.Sprintf("[%s]%s[-]", color, key)
			title = strings.Replace(title, key, hotkeyText, 1)
		}
		area := mi.HotKeys[0]
		switch area {
		case "/":
			area = "root"
		case "~":
			area = "home"
		case "±":
			area = "size"
		}
		title = fmt.Sprintf(`["%s"]%s[""]`, area, title)
		sb.WriteString(title)
		sb.WriteString(separator)
	}
	fullText := sb.String()
	trimmedText := fullText[:sb.Len()-len(separator)]
	return trimmedText
}

func (b *bottom) highlighted(added, _, _ []string) {
	if len(added) == 0 {
		return
	}

	menuItems := b.altMenuItems
	if b.isCtrl {
		menuItems = b.getCtrlMenuItems()
	}

	region := added[0]
	for _, mi := range menuItems {
		if mi.HotKeys[0] == region && mi.Action != nil {
			mi.Action()
			return
		}
	}
}

var archiveAction = func() {
}

var osExit = os.Exit

func (b *bottom) getCtrlMenuItems() []ftui.MenuItem {
	// Unfortunately, most terminals do not send standalone CTRL key event,
	// so we can't handle it to show an alternative menu when CTRL is pressed.
	return []ftui.MenuItem{
		{
			Title:       "Archive",
			HotKeys:     []string{"A"},
			Action:      archiveAction,
			IsAltHotkey: true,
		},
		{
			Title:       "Stage",
			HotKeys:     []string{"S"},
			IsAltHotkey: true,
		},
		{
			Title:       "Commit",
			HotKeys:     []string{"C"},
			IsAltHotkey: true,
		},
		{
			Title:       "Push",
			HotKeys:     []string{"P"},
			IsAltHotkey: true,
		},
	}
}

func (b *bottom) getFKMenuItems() []ftui.MenuItem {
	return []ftui.MenuItem{
		{Title: "F1·Help", HotKeys: []string{"F1"}, Action: func() {}},
		{Title: "F2·Menu", HotKeys: []string{"F2"}, Action: func() {}},
		{Title: "F3·View", HotKeys: []string{"F3"}, Action: func() {}},
		{Title: "F4·Edit", HotKeys: []string{"F4"}, Action: func() {}},
		{Title: "F5·Copy", HotKeys: []string{"F5"}, Action: func() {}},
		{Title: "F6·Rename", HotKeys: []string{"F6"}, Action: func() {}},
		{Title: "F7·Create", HotKeys: []string{"F7"}, Action: func() {}},
		{Title: "F8·Delete", HotKeys: []string{"F8"}, Action: func() {}},
	}
}

func (b *bottom) getAltMenuItems() []ftui.MenuItem {
	return []ftui.MenuItem{
		{Title: "Exit", HotKeys: []string{"x"}, Action: func() { b.nav.app.Stop(); osExit(0) }, IsAltHotkey: true},
		{Title: "Go", HotKeys: []string{"o"}, Action: func() {}, IsAltHotkey: true},
		{Title: "/root", HotKeys: []string{"/"}, Action: func() {}, IsAltHotkey: true},
		{Title: "~Home", HotKeys: []string{"~"}, Action: func() {}, IsAltHotkey: true},
		{Title: "Favorites", HotKeys: []string{"F"}, Action: func() {}, IsAltHotkey: true},
		{Title: "Bookmarks", HotKeys: []string{"B"}, Action: func() {}, IsAltHotkey: true},
		{Title: "Lists", HotKeys: []string{"L"}, Action: func() {}, IsAltHotkey: true},
		{Title: "Masks", HotKeys: []string{"M"}, Action: func() {}, IsAltHotkey: true},
		{Title: "±Size", HotKeys: []string{"±"}, Action: func() {}, IsAltHotkey: true},
		{Title: "Git", HotKeys: []string{"G"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "Previewer", HotKeys: []string{"P"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "Copy", HotKeys: []string{"F5", "C"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "Rename", HotKeys: []string{"F6", "R"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "Delete", HotKeys: []string{"F8", "D"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "View", HotKeys: []string{"V"}, Action: func() {}, IsAltHotkey: true},
		//{Title: "Edit", HotKeys: []string{"E"}, Action: func() {}, IsAltHotkey: true},
	}
}
