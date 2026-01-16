package filetug

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/datatug/filetug/pkg/ftstate"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type files struct {
	*boxed
	table *tview.Table
	rows  *FileRows
	nav   *Navigator
	filterTabs
	filter          Filter
	currentFileName string
}

//func (f *files) Clear() {
//	f.extTable.Clear()
//}

//func (f *files) Draw(screen tcell.Screen) {
//	//f.selectCurrentFile()
//	f.boxed.Draw(screen)
//}

func (f *files) SetRows(rows *FileRows) {
	f.table.Select(0, 0)
	rows.SetFilter(f.filter)
	f.rows = rows
	f.table.SetContent(rows)
	if f.currentFileName != "" {
		f.selectCurrentFile()
	}
	f.table.ScrollToBeginning()
}

func (f *files) SetFilter(filter Filter) {
	f.rows.SetFilter(filter)
}

func (f *files) selectCurrentFile() {
	if f.currentFileName == "" || f.rows == nil {
		return
	}
	for i, entry := range f.rows.AllEntries {
		if entry.Name() == f.currentFileName {
			row, _ := f.table.GetSelection()
			if row != i+1 {
				f.table.Select(i+1, 0)
			}
			return
		}
	}
}

func (f *files) SetCurrentFile(name string) {
	f.currentFileName = name
	f.selectCurrentFile()
}

func (f *files) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	table := f.table
	if string(event.Rune()) == " " {
		row, _ := table.GetSelection()
		cell := table.GetCell(row, 0)

		if strings.HasPrefix(cell.Text, " ") {
			cell.SetText("✓" + strings.TrimPrefix(cell.Text, " "))
		} else {
			cell.SetText(" " + strings.TrimPrefix(cell.Text, "✓"))
		}
		return nil
	}
	switch event.Key() {
	case tcell.KeyLeft:
		f.nav.app.SetFocus(f.nav.dirsTree)
		return nil
	case tcell.KeyRight:
		f.nav.app.SetFocus(f.nav.right)
		return nil
	case tcell.KeyUp:
		row, _ := table.GetSelection()
		if row == 0 {
			if f.nav.o.moveFocusUp != nil {
				f.nav.o.moveFocusUp(table)
				return nil
			}
		}
		return event
	case tcell.KeyEnter:
		row, _ := table.GetSelection()
		nameCell := table.GetCell(row, 0)
		switch ref := nameCell.GetReference().(type) {
		case DirEntry:
			f.nav.goDir(ref.Path)
			return nil
		default:
			return event
		}
	default:
		return event
	}
}

type filterTabs struct {
	nav       *Navigator
	filesTab  *Tab
	dirsTab   *Tab
	hiddenTab *Tab
}

func newFilterTabs(nav *Navigator) filterTabs {
	return filterTabs{
		nav:       nav,
		filesTab:  &Tab{Title: "Files", Hotkey: 'e', Checked: true},
		dirsTab:   &Tab{Title: "Dirs", Hotkey: 'r', Checked: false},
		hiddenTab: &Tab{Title: "Hidden", Hotkey: 'H', Checked: false},
	}
}

func newFiles(nav *Navigator) *files {
	table := tview.NewTable()
	//extTable := sticky.NewTable([]sticky.Column{
	//	{
	//		Name:      "Name",
	//		Expansion: 1,
	//		MinWidth:  20,
	//	},
	//	{
	//		Name:       "Size",
	//		FixedWidth: 6,
	//	},
	//	{
	//		Name:       "Modified",
	//		FixedWidth: 10,
	//	},
	//})
	flex := tview.NewFlex()
	flex.AddItem(table, 0, 1, true)

	tabs := newFilterTabs(nav)

	f := &files{
		nav:   nav,
		table: table,
		boxed: newBoxed(
			flex,
			WithLeftBorder(0, -1),
			WithRightBorder(0, +1),
			WithTabs(tabs.filesTab, tabs.dirsTab, tabs.hiddenTab),
		),
		filterTabs: tabs,
	}
	table.SetSelectable(true, false)
	//table.SetFixed(1, 0)
	table.SetInputCapture(f.inputCapture)
	table.SetFocusFunc(f.focus)
	table.SetBlurFunc(f.blur)

	table.SetSelectionChangedFunc(f.selectionChanged)
	nav.filesSelectionChangedFunc = f.selectionChangedNavFunc
	f.blur()
	return f
}

func (f *files) focus() {
	f.nav.activeCol = 1
	f.nav.right.SetContent(f.nav.previewer)
	f.table.SetSelectedStyle(theme.FocusedSelectedTextStyle)
}

func (f *files) blur() {
	f.table.SetSelectedStyle(theme.BlurredSelectedTextStyle)
}

// selectionChangedNavFunc: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *files) selectionChangedNavFunc(row, _ int) {
	cell := f.table.GetCell(row, 0)
	name := cell.Text[1:]
	fullName := filepath.Join(f.nav.current.dir, name)
	f.nav.right.SetContent(f.nav.previewer)
	f.nav.previewer.PreviewFile(name, fullName)
}

// selectionChanged: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *files) selectionChanged(row, _ int) {
	if row == 0 {
		f.nav.previewer.textView.SetText("Selected dir: " + f.nav.current.dir)
		f.nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
		return
	}
	cell := f.table.GetCell(row, 0)
	ref := cell.GetReference()
	if ref == nil {
		f.nav.previewer.SetText("cell has no reference")
		return
	}

	dirEntry := ref.(DirEntry)
	fullName := filepath.Join(dirEntry.Path, dirEntry.Name())
	f.rememberCurrent(fullName)

	stat, err := os.Stat(fullName)
	if err != nil {
		f.nav.previewer.SetErr(err)
		return
	}

	if stat.IsDir() {
		f.nav.previewer.SetText("Directory: " + fullName)
		return
	}

	f.nav.previewer.PreviewFile("", fullName)
}

func (f *files) rememberCurrent(fullName string) {
	_, f.currentFileName = path.Split(fullName)
	ftstate.SaveCurrentFileName(f.currentFileName)
}
