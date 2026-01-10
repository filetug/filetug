package filetug

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/datatug/filetug/pkg/sticky"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var _ sticky.Records = (*fsRecords)(nil)

type fsRecords struct {
	nodePath   string
	dirEntries []os.DirEntry
}

func (r fsRecords) Count() int {
	return len(r.dirEntries)
}

func (r fsRecords) GetCell(row, _ int, colName string) *tview.TableCell {
	dirEntry := r.dirEntries[row]
	var cell *tview.TableCell
	name := dirEntry.Name()
	if colName == "Name" {
		if dirEntry.IsDir() {
			cell = tview.NewTableCell(" 📁" + name)
		} else {
			cell = tview.NewTableCell("   " + name)
		}
	} else {
		info, err := dirEntry.Info()
		if err != nil {
			return tview.NewTableCell(err.Error()).SetBackgroundColor(tcell.ColorRed)
		}
		switch colName {
		case "Size":
			cell = tview.NewTableCell(strconv.FormatInt(info.Size(), 10)).SetAlign(tview.AlignRight)
		case "Modified":
			var s string
			if modTime := info.ModTime(); info.ModTime().After(time.Now().Add(24 * time.Hour)) {
				s = modTime.Format("15:04:05")
			} else {
				s = modTime.Format("2006-01-02")
			}
			cell = tview.NewTableCell(s)
		default:
			return nil
		}
	}
	color := GetColorByFileExt(name)
	cell.SetTextColor(color)
	cell.SetReference(fsutils.ExpandHome(path.Join(r.nodePath, name)))
	return cell
}

func newFiles(nav *Navigator) *sticky.Table {
	files := sticky.NewTable([]sticky.Column{
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
			nav.app.SetFocus(nav.dirsTree)
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
	nav.filesFocusFunc = func() {
		files.SetBorderColor(Style.FocusedBorderColor)
		nav.activeCol = 1
	}

	files.SetBlurFunc(func() {
		files.SetBorderColor(Style.BlurBorderColor)
	})
	nav.filesBlurFunc = func() {
		files.SetBorderColor(Style.BlurBorderColor)
	}

	files.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			nav.previewer.textView.SetText("Selected dir: " + nav.currentDir)
			nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
			return
		}
		cell := files.GetCell(row, 0)
		ref := cell.GetReference()
		if ref == nil {
			nav.previewer.SetText("cell has no reference")
			return
		}
		fullName := ref.(string)
		stat, err := os.Stat(fullName)
		if err != nil {
			nav.previewer.SetErr(err)
			return
		}
		if stat.IsDir() {
			nav.previewer.SetText("Directory: " + fullName)
			return
		}
		nav.previewer.PreviewFile("", fullName)
	})
	nav.filesSelectionChangedFunc = func(row, column int) {
		if row == 0 {
			nav.previewer.textView.SetText("Selected dir: " + nav.currentDir)
			nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
			return
		}
		cell := files.GetCell(row, 0)
		name := cell.Text[1:]
		fullName := filepath.Join(nav.currentDir, name)
		nav.previewer.PreviewFile(name, fullName)
	}
	return files
}
