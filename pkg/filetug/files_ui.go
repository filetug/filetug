package filetug

import (
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/themes"
)

// onStoreChange is called when the store changes. It clears the table and
// displays a loading indicator with animation.
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
	// We only start animation if we have a real app running.
	// In tests, NewNavigator(app) uses a sync QueueUpdateDraw which we check in doLoadingAnimation.
	go func() {
		f.doLoadingAnimation(progressCell)
	}()
}

// doLoadingAnimation displays an animated loading progress bar.
func (f *filesPanel) doLoadingAnimation(loading *tview.TableCell) {
	// Simple heuristic: if we are in a test (no real app), don't loop
	// Using a shorter sleep to avoid hanging tests that expect synchronous updates
	for f.table.GetCell(1, 0) == loading {
		q, r := f.loadingProgress/len(spinner), f.loadingProgress%len(spinner)
		progressBar := strings.Repeat("█", q) + string(spinner[r])
		if f.nav != nil && f.nav.app != nil {
			f.nav.app.QueueUpdateDraw(func() {
				loading.SetText(progressBar)
			})
		}
		f.loadingProgress += 1
		time.Sleep(10 * time.Millisecond)
	}
}

// SetRows sets the file rows to display and applies the current filter.
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
		if f.nav != nil && f.nav.app != nil {
			f.nav.app.QueueUpdateDraw(func() {
				f.table.ScrollToBeginning()
			})
		}
	}()

}

// SetFilter updates the filter applied to the file rows.
func (f *filesPanel) SetFilter(filter ftui.Filter) {
	f.rows.SetFilter(filter)
}

// inputCapture handles keyboard input for the files panel.
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
	case tcell.KeyRune:
		return f.nav.globalNavInputCapture(event)
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
		dirContext := files.NewDirContext(f.nav.store, fullPath, nil)
		f.nav.goDir(dirContext)
		return nil
	default:
		return event
	}
}

// focus is called when the files panel gains focus.
func (f *filesPanel) focus() {
	f.nav.activeCol = 1
	f.nav.right.SetContent(f.nav.previewer)
	f.table.SetSelectedStyle(themes.CurrentTheme.FocusedSelectedTextStyle())
}

// blur is called when the files panel loses focus.
func (f *filesPanel) blur() {
	f.table.SetSelectedStyle(themes.CurrentTheme.BlurredSelectedTextStyle())
}
