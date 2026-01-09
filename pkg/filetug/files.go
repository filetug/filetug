package filetug

import (
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newFiles(nav *Navigator) *tview.Table {
	files := tview.NewTable()
	files.SetSelectable(true, false)
	files.SetFixed(1, 1)
	files.SetBorder(true)
	files.SetBorderColor(Style.BlurBorderColor)
	files.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if string(event.Rune()) == " " {
			row, _ := files.GetSelection()
			cell := files.GetCell(row, 0)

			if strings.HasPrefix(cell.Text, " ") {
				cell.SetText("✓" + strings.TrimPrefix(cell.Text, " "))
			} else {
				cell.SetText(" " + strings.TrimPrefix(cell.Text, "✓"))
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyLeft:
			nav.app.SetFocus(nav.dirs)
			return nil
		case tcell.KeyRight:
			nav.app.SetFocus(nav.previewer)
			return nil
		case tcell.KeyUp:
			row, _ := files.GetSelection()
			if row == 0 {
				nav.o.moveFocusUp(files)
				return nil
			}
			return event
		default:
			return event
		}
	})
	files.SetFocusFunc(func() {
		files.SetBorderColor(Style.FocusedBorderColor)
		nav.activeCol = 1
	})
	files.SetBlurFunc(func() {
		files.SetBorderColor(Style.BlurBorderColor)
	})
	files.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			nav.previewer.textView.SetText("Selected dir: " + nav.currentDir)
			nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
			return
		}
		cell := files.GetCell(row, 0)
		name := cell.Text[1:]
		fullName := filepath.Join(nav.currentDir, name)
		nav.previewer.PreviewFile(name, fullName)
	})
	return files
}
