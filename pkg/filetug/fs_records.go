package filetug

import (
	"os"
	"path"
	"strconv"
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
	infos      []os.FileInfo
	err        error
}

func NewDirRecords(nodePath string, dirEntries []os.DirEntry) sticky.Records {
	return &fsRecords{
		nodePath:   nodePath,
		dirEntries: dirEntries,
		infos:      make([]os.FileInfo, len(dirEntries)),
	}
}

func (r fsRecords) Count() int {
	if r.err != nil {
		return 1
	}
	if len(r.dirEntries) == 0 {
		return 1
	}
	return len(r.dirEntries)
}

func (r fsRecords) GetCell(row, _ int, colName string) *tview.TableCell {
	if r.err != nil {
		if colName == "Name" {
			return tview.NewTableCell(" 📁" + r.err.Error()).SetTextColor(tcell.ColorOrangeRed)
		}
		return nil
	}
	if len(r.dirEntries) == 0 {
		if row == 0 && colName == "Name" {
			return tview.NewTableCell("[::i]No entries[::-]").SetTextColor(tcell.ColorGray)
		}
		return nil
	}
	dirEntry := r.dirEntries[row]
	var cell *tview.TableCell
	name := dirEntry.Name()
	if colName == "Name" {
		if dirEntry.IsDir() {
			cell = tview.NewTableCell(" 📁" + name)
		} else {
			cell = tview.NewTableCell(" 📄" + name)
		}
	} else {
		fi := r.infos[row]
		if fi == nil {
			var err error
			fi, err = dirEntry.Info()
			if err != nil {
				return tview.NewTableCell(err.Error()).SetBackgroundColor(tcell.ColorRed)
			}
			r.infos[row] = fi
		}

		switch colName {
		case "Size":
			cell = tview.NewTableCell(strconv.FormatInt(fi.Size(), 10)).SetAlign(tview.AlignRight)
		case "Modified":
			var s string
			if modTime := fi.ModTime(); fi.ModTime().After(time.Now().Add(24 * time.Hour)) {
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
