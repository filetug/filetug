package filetug

import (
	"fmt"
	"os"
	"strings"

	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type bottom struct {
	*tview.TextView
	nav       *Navigator
	menuItems []ftui.MenuItem
	isCtrl    bool
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

	b.menuItems = b.getAltMenuItems()
	b.render()

	return b
}

func (b *bottom) render() {
	const separator = "┊"
	var sb strings.Builder
	sb.WriteString("[white]Alt[-]+: ")
	for _, mi := range b.menuItems {
		title := mi.Title
		for _, key := range mi.HotKeys {
			hotkeyText := fmt.Sprintf("[%s]%s[-]", sneatv.CurrentTheme.HotkeyColor, key)
			title = strings.Replace(title, key, hotkeyText, 1)
		}
		title = fmt.Sprintf(`["%s"]%s[""]`, mi.HotKeys[0], title)
		sb.WriteString(title)
		sb.WriteString(separator)
	}
	fullText := sb.String()
	trimmedText := fullText[:sb.Len()-len(separator)]
	b.SetText(trimmedText)
}

func (b *bottom) highlighted(added, _, _ []string) {
	if len(added) == 0 {
		return
	}

	menuItems := b.menuItems
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
			Title:   "Archive",
			HotKeys: []string{"A"},
			Action:  archiveAction,
		},
		{
			Title:   "Stage",
			HotKeys: []string{"S"},
		},
		{
			Title:   "Commit",
			HotKeys: []string{"C"},
		},
		{
			Title:   "Push",
			HotKeys: []string{"P"},
		},
	}
}

func (b *bottom) getAltMenuItems() []ftui.MenuItem {
	return []ftui.MenuItem{
		{Title: "F1Help", HotKeys: []string{"F1"}, Action: func() {}},
		{Title: "Exit", HotKeys: []string{"x"}, Action: func() { b.nav.stopApp(); osExit(0) }},
		{Title: "Go", HotKeys: []string{"o"}, Action: func() {}},
		//{Title: "/root", HotKeys: []string{"/"}, Action: func() {}},
		{Title: "~Home", HotKeys: []string{"H", "~"}, Action: func() {}},
		{Title: "Favorites", HotKeys: []string{"F"}, Action: func() {}},
		{Title: "Bookmarks", HotKeys: []string{"B"}, Action: func() {}},
		{Title: "Lists", HotKeys: []string{"L"}, Action: func() {}},
		//{Title: "Previewer", HotKeys: []string{"P"}, Action: func() {}},
		{Title: "Masks", HotKeys: []string{"M"}, Action: func() {}},
		{Title: "Git", HotKeys: []string{"G"}, Action: func() {}},
		{Title: "Copy", HotKeys: []string{"F5", "C"}, Action: func() {}},
		{Title: "Rename", HotKeys: []string{"F6", "R"}, Action: func() {}},
		{Title: "Delete", HotKeys: []string{"F8", "D"}, Action: func() {}},
		{Title: "View", HotKeys: []string{"V"}, Action: func() {}},
		{Title: "Edit", HotKeys: []string{"E"}, Action: func() {}},
	}
}
