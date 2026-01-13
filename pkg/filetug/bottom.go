package filetug

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type bottom struct {
	*tview.TextView
}

type MenuItem struct {
	Title   string
	HotKeys []string
	Action  func()
}

func newBottom() *bottom {
	b := &bottom{
		TextView: tview.NewTextView().SetDynamicColors(true),
	}

	b.SetTextColor(tcell.ColorSlateGray)

	menuItems := []MenuItem{
		{
			Title:   "F1-ç≈Help",
			HotKeys: []string{"F1"},
			Action:  func() {},
		},
		{
			Title:   "GoTo",
			HotKeys: []string{"G"},
			Action:  func() {},
		},
		{
			Title:   "Favorites",
			HotKeys: []string{"F"},
			Action:  func() {},
		},
		{
			Title:   "Bookmarks",
			HotKeys: []string{"B"},
			Action:  func() {},
		},
		{
			Title:   "Previewer",
			HotKeys: []string{"P"},
			Action:  func() {},
		},
		{
			Title:   "Copy",
			HotKeys: []string{"F5", "C"},
			Action:  func() {},
		},
		{
			Title:   "Move",
			HotKeys: []string{"F6", "M"},
			Action:  func() {},
		},
		{
			Title:   "Delete",
			HotKeys: []string{"F8", "D"},
			Action:  func() {},
		},
		{
			Title:   "View",
			HotKeys: []string{"V"},
			Action:  func() {},
		},
		{
			Title:   "Edit",
			HotKeys: []string{"E"},
			Action:  func() {},
		},
		{
			Title:   "Exit",
			HotKeys: []string{"x"},
			Action:  func() {},
		},
	}

	const separator = " | "
	var sb strings.Builder
	for _, mi := range menuItems {
		title := mi.Title
		for _, key := range mi.HotKeys {
			title = strings.Replace(title, key, "[white]"+key+"[-]", 1)
		}
		sb.WriteString(title)
		sb.WriteString(separator)
	}
	b.SetText(sb.String()[:sb.Len()-len(separator)])
	return b
}
