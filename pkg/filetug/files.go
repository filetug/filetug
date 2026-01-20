package filetug

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/datatug/filetug/pkg/filetug/ftstate"
	"github.com/datatug/filetug/pkg/filetug/ftui"
	"github.com/datatug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

//func (f *filesPanel) Clear() {
//	f.extTable.Clear()
//}

//func (f *filesPanel) Draw(screen tcell.Screen) {
//	//f.selectCurrentFile()
//	f.Boxed.Draw(screen)
//}

func (f *filesPanel) onStoreChange() {
	f.loadingProgress = 0
	f.table.SetContent(nil)
	f.table.Clear()
	loadingCell := tview.NewTableCell("Loading...")
	loadingCell = loadingCell.SetTextColor(tcell.ColorLightGray)
	progressCell := tview.NewTableCell("")
	progressCell = progressCell.SetTextColor(tcell.ColorDarkGrey)
	f.table.SetCell(0, 0, loadingCell)
	f.table.SetCell(1, 0, progressCell)
	f.table.SetSelectable(false, false)
	go func() {
		go f.doLoadingAnimation(progressCell)
	}()
}

func (f *filesPanel) doLoadingAnimation(loading *tview.TableCell) {
	time.Sleep(10 * time.Millisecond)
	if f.table.GetCell(1, 0) == loading {
		q, r := f.loadingProgress/len(spinner), f.loadingProgress%len(spinner)
		progressBar := strings.Repeat("█", q) + string(spinner[r])
		if f.nav != nil && f.nav.queueUpdateDraw != nil {
			f.nav.queueUpdateDraw(func() {
				loading.SetText(progressBar)
			})
		} else {
			loading.SetText(progressBar)
		}
		f.loadingProgress += 1
		f.doLoadingAnimation(loading)
	}
}

func (f *filesPanel) SetRows(rows *FileRows, showDirs bool) {
	f.table.Select(0, 0)
	f.filter.ShowDirs = showDirs
	rows.SetFilter(f.filter)
	f.rows = rows
	f.table.SetContent(rows)
	if f.currentFileName != "" {
		f.selectCurrentFile()
	}
	go func() {
		time.Sleep(time.Millisecond)
		if f.nav != nil && f.nav.queueUpdateDraw != nil {
			f.nav.queueUpdateDraw(func() {
				f.table.ScrollToBeginning()
			})
		}
	}()

}

func (f *filesPanel) SetFilter(filter ftui.Filter) {
	f.rows.SetFilter(filter)
}

func (f *filesPanel) selectCurrentFile() {
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

func (f *filesPanel) SetCurrentFile(name string) {
	f.currentFileName = name
	f.selectCurrentFile()
}

func (f *filesPanel) inputCapture(event *tcell.EventKey) *tcell.EventKey {
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
		f.nav.setAppFocus(f.nav.dirsTree)
		return nil
	case tcell.KeyRight:
		f.nav.setAppFocus(f.nav.right)
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
	filesTab  *sneatv.PanelTab
	dirsTab   *sneatv.PanelTab
	hiddenTab *sneatv.PanelTab
}

func newFilterTabs(nav *Navigator) filterTabs {
	return filterTabs{
		nav:       nav,
		filesTab:  &sneatv.PanelTab{Title: "Files", Hotkey: 'e', Checked: true},
		dirsTab:   &sneatv.PanelTab{Title: "Dirs", Hotkey: 'r', Checked: false},
		hiddenTab: &sneatv.PanelTab{Title: "Hidden", Hotkey: 'H', Checked: false},
	}
}

func newFiles(nav *Navigator) *filesPanel {
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
	//table.SetFixed(1, 0)
	table.SetInputCapture(f.inputCapture)
	table.SetFocusFunc(f.focus)
	table.SetBlurFunc(f.blur)

	table.SetSelectionChangedFunc(f.selectionChanged)
	nav.filesSelectionChangedFunc = f.selectionChangedNavFunc
	f.blur()
	return f
}

func (f *filesPanel) focus() {
	f.nav.activeCol = 1
	f.nav.right.SetContent(f.nav.previewer)
	f.table.SetSelectedStyle(sneatv.CurrentTheme.FocusedSelectedTextStyle)
}

func (f *filesPanel) blur() {
	f.table.SetSelectedStyle(sneatv.CurrentTheme.BlurredSelectedTextStyle)
}

// selectionChangedNavFunc: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *filesPanel) selectionChangedNavFunc(row, _ int) {
	cell := f.table.GetCell(row, 0)
	name := cell.Text[1:]
	fullName := filepath.Join(f.nav.current.dir, name)
	f.nav.right.SetContent(f.nav.previewer)
	f.nav.previewer.PreviewFile(name, fullName)
}

// selectionChanged: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *filesPanel) selectionChanged(row, _ int) {
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

func (f *filesPanel) rememberCurrent(fullName string) {
	_, f.currentFileName = path.Split(fullName)
	ftstate.SaveCurrentFileName(f.currentFileName)
}
