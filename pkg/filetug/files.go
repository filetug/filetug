package filetug

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

var _ browser = (*filesPanel)(nil)

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

func (f *filesPanel) GetCurrentEntry() files.EntryWithDirPath {
	row, _ := f.table.GetSelection()
	if row >= len(f.rows.VisibleEntries) {
		return nil
	}
	entry := f.rows.VisibleEntries[row]
	if entry.DirPath() == "" {
		if f.rows.Dir == nil {
			_, _ = fmt.Fprintf(os.Stderr, "files panel missing dir path for entry %q\n", entry.Name())
			return nil
		}
		entry = files.NewEntryWithDirPath(entry, f.rows.Dir.Path)
	}

	return entry
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

func (f *filesPanel) updateGitStatuses(ctx context.Context, dirContext *DirContext) {
	if f.nav == nil || f.rows == nil || dirContext == nil {
		return
	}
	if f.nav.store.RootURL().Scheme != "file" {
		return
	}
	repoRoot := gitutils.GetRepositoryRoot(dirContext.Path)
	if repoRoot == "" {
		return
	}
	repo, err := git.PlainOpen(repoRoot)
	if err != nil {
		return
	}

	rows := f.rows
	table := f.table
	queueUpdateDraw := f.nav.queueUpdateDraw
	for _, entry := range rows.AllEntries {
		entry := entry
		fullPath := entry.FullName()
		isDir := entry.IsDir()
		if !isDir {
			isDir = rows.isSymlinkToDir(entry)
		}

		go func() {
			status := f.nav.getGitStatus(ctx, repo, fullPath, isDir)
			if status == nil {
				return
			}
			statusText := f.nav.gitStatusText(status, fullPath, isDir)
			updated := rows.SetGitStatusText(fullPath, statusText)
			if !updated || queueUpdateDraw == nil {
				return
			}
			queueUpdateDraw(func() {
				if f.rows != rows {
					return
				}
				table.SetContent(rows)
			})
		}()
	}
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
			trimmed := strings.TrimPrefix(cell.Text, " ")
			cell.SetText("✓" + trimmed)
		} else {
			trimmed := strings.TrimPrefix(cell.Text, "✓")
			cell.SetText(" " + trimmed)
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
		refValue := nameCell.GetReference()
		entry, ok := refValue.(files.EntryWithDirPath)
		if !ok || entry == nil {
			return event
		}
		isDir := entry.IsDir()
		if !isDir && f.rows != nil {
			isDir = f.rows.isSymlinkToDir(entry)
		}
		if !isDir { // TODO: Open file for view?
			return event
		}
		fullPath := entry.FullName()
		f.nav.goDir(fullPath)
		return nil
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
	entry := f.entryFromRow(row)
	if entry == nil {
		return
	}
	f.updatePreviewForEntry(entry)
}

// selectionChanged: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *filesPanel) selectionChanged(row, _ int) {
	entry := f.entryFromRow(row)
	if entry == nil {
		if f.nav != nil && f.nav.previewer != nil {
			f.nav.previewer.SetText("cell has no reference")
		}
		return
	}
	f.updatePreviewForEntry(entry)
}

func (f *filesPanel) rememberCurrent(fullName string) {
	_, f.currentFileName = path.Split(fullName)
	ftstate.SaveCurrentFileName(f.currentFileName)
}

func (f *filesPanel) entryFromRow(row int) files.EntryWithDirPath {
	if f.table == nil {
		return nil
	}
	cell := f.table.GetCell(row, 0)
	ref := cell.GetReference()
	if ref == nil {
		return nil
	}
	entry, ok := ref.(files.EntryWithDirPath)
	if !ok || entry == nil {
		return nil
	}
	return entry
}

func (f *filesPanel) updatePreviewForEntry(entry files.EntryWithDirPath) {
	nav := f.nav
	if nav == nil {
		return
	}
	isDir := entry.IsDir()
	if !isDir && f.rows != nil {
		isDir = f.rows.isSymlinkToDir(entry)
	}
	if isDir {
		f.showDirSummary(entry)
		return
	}

	if nav.right != nil && nav.previewer != nil {
		content := nav.previewer
		nav.right.SetContent(content)
	}
	fullName := entry.FullName()
	f.rememberCurrent(fullName)
	if nav.previewer == nil {
		return
	}
	nav.previewer.PreviewEntry(entry)
}

func (f *filesPanel) showDirSummary(entry files.EntryWithDirPath) {
	nav := f.nav
	if nav == nil || nav.dirSummary == nil || nav.right == nil {
		return
	}
	content := nav.dirSummary
	nav.right.SetContent(content)

	dirPath := entry.DirPath()
	if entry.IsDir() {
		dirPath = entry.FullName()
	} else if f.rows != nil && f.rows.isSymlinkToDir(entry) {
		dirPath = entry.FullName()
	}

	if nav.store == nil {
		nav.dirSummary.SetDir(dirPath, nil)
		return
	}
	ctx := context.Background()
	entries, err := nav.store.ReadDir(ctx, dirPath)
	if err != nil {
		nav.dirSummary.SetDir(dirPath, nil)
		return
	}
	sortedEntries := sortDirChildren(entries)
	nav.dirSummary.SetDir(dirPath, sortedEntries)
}
