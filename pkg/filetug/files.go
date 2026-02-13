package filetug

import (
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/rivo/tview"
)

var _ browser = (*filesPanel)(nil)

// filesPanel is the main panel that displays the list of files and directories.
// It integrates with git status display, filtering, and preview functionality.
type filesPanel struct {
	*sneatv.Boxed
	table *tview.Table
	rows  *FileRows
	nav   *Navigator
	filterTabs
	filter          ftui.Filter
	currentFileName string
	loadingProgress int
}

// filterTabs holds the tab definitions for filtering files, directories, and hidden files.
type filterTabs struct {
	nav       *Navigator
	filesTab  *sneatv.PanelTab
	dirsTab   *sneatv.PanelTab
	hiddenTab *sneatv.PanelTab
}

// newFilterTabs creates a new set of filter tabs with default settings.
func newFilterTabs(nav *Navigator) filterTabs {
	return filterTabs{
		nav:       nav,
		filesTab:  &sneatv.PanelTab{Title: "Files", Hotkey: 'e', Checked: true},
		dirsTab:   &sneatv.PanelTab{Title: "Dirs", Hotkey: 'r', Checked: false},
		hiddenTab: &sneatv.PanelTab{Title: "Hidden", Hotkey: 'H', Checked: false},
	}
}

// newFiles creates a new files panel with the given navigator.
func newFiles(nav *Navigator) *filesPanel {
	table := tview.NewTable()
	flex := tview.NewFlex()
	flex.AddItem(table, 0, 1, true)

	tabs := newFilterTabs(nav)

	f := &filesPanel{
		nav:   nav,
		table: table,
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(0, -1),
			sneatv.WithRightBorder(0, +1),
			sneatv.WithTabs(tabs.filesTab, tabs.dirsTab, tabs.hiddenTab),
		),
		filterTabs: tabs,
	}
	table.SetSelectable(true, false)
	table.SetInputCapture(f.inputCapture)
	table.SetFocusFunc(f.focus)
	table.SetBlurFunc(f.blur)

	table.SetSelectionChangedFunc(f.selectionChanged)
	nav.filesSelectionChangedFunc = f.selectionChangedNavFunc
	f.blur()
	return f
}
