package filetug

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/datatug/filetug/pkg/sticky"
	"github.com/gdamore/tcell/v2"
)

type files struct {
	*sticky.Table
	nav   *Navigator
	boxed *boxed
}

func (f *files) Draw(screen tcell.Screen) {
	f.boxed.Draw(screen)
}

func (f *files) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	table := f.Table
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
		f.nav.app.SetFocus(f.nav.previewer)
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
	default:
		return event
	}
}

func newFiles(nav *Navigator) *files {
	table := sticky.NewTable([]sticky.Column{
		{
			Name:      "Name",
			Expansion: 1,
			MinWidth:  20,
		},
		{
			Name:       "Size",
			FixedWidth: 6,
		},
		{
			Name:       "Modified",
			FixedWidth: 10,
		},
	})
	f := &files{
		nav:   nav,
		Table: table,
		boxed: newBoxed(
			table,
			WithLeftBorder(0, -1),
			WithRightBorder(0, +1),
		),
	}
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetInputCapture(f.inputCapture)
	table.SetFocusFunc(func() {
		nav.activeCol = 1
	})
	nav.filesFocusFunc = func() {
		nav.activeCol = 1
	}

	table.SetSelectionChangedFunc(f.selectionChanged)
	nav.filesSelectionChangedFunc = f.selectionChangedNavFunc
	return f
}

// selectionChangedNavFunc: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *files) selectionChangedNavFunc(row, _ int) {
	if row == 0 {
		f.nav.previewer.textView.SetText("Selected dir: " + f.nav.currentDir)
		f.nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
		return
	}
	cell := f.GetCell(row, 0)
	name := cell.Text[1:]
	fullName := filepath.Join(f.nav.currentDir, name)
	f.nav.previewer.PreviewFile(name, fullName)
}

// selectionChanged: TODO: is it a duplicate of selectionChangedNavFunc?
func (f *files) selectionChanged(row, _ int) {
	if row == 0 {
		f.nav.previewer.textView.SetText("Selected dir: " + f.nav.currentDir)
		f.nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
		return
	}
	cell := f.GetCell(row, 0)
	ref := cell.GetReference()
	if ref == nil {
		f.nav.previewer.SetText("cell has no reference")
		return
	}
	fullName := ref.(string)
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
