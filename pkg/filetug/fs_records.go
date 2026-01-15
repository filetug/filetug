package filetug

import (
	"os"
	"path"
	"time"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//var _ sticky.Records = (*FileRows)(nil)

var _ tview.TableContent = (*FileRows)(nil)

type Filter struct {
	Extensions []string
}

func (f Filter) IsEmpty() bool {
	return len(f.Extensions) == 0
}

func (f Filter) IsVisibleByDirEntry(entry os.DirEntry) bool {
	if len(f.Extensions) == 0 {
		return true
	}
	for _, ext := range f.Extensions {
		if path.Ext(entry.Name()) == ext {
			return true
		}
	}
	return false
}

type FileRows struct {
	tview.TableContentReadOnly
	NodePath       string
	AllEntries     []os.DirEntry
	VisibleEntries []os.DirEntry
	Infos          []os.FileInfo
	VisualInfos    []os.FileInfo
	Err            error
	filter         Filter
}

//func (r *FileRows) SetSelected(row int) {
//	if row == 0 {
//		r.selected = ""
//	}
//	r.selected = r.AllEntries[row-1].Name()
//}

func (r *FileRows) SetFilter(filter Filter) {
	r.filter = filter
	r.applyFilter()
}

func (r *FileRows) applyFilter() {
	r.VisibleEntries = make([]os.DirEntry, 0, len(r.AllEntries))
	r.VisualInfos = make([]os.FileInfo, 0, len(r.VisibleEntries))
	for i, entry := range r.AllEntries {
		if r.filter.IsVisibleByDirEntry(entry) {
			r.VisibleEntries = append(r.VisibleEntries, entry)
			r.VisualInfos = append(r.VisualInfos, r.Infos[i])
		}
	}
}

func (r *FileRows) GetRowCount() int {
	return len(r.VisibleEntries)
}

func (r *FileRows) GetColumnCount() int {
	return 3
}

func NewFileRows(nodePath string, dirEntries []os.DirEntry) *FileRows {
	return &FileRows{
		NodePath:       nodePath,
		AllEntries:     dirEntries,
		VisibleEntries: dirEntries,
		Infos:          make([]os.FileInfo, len(dirEntries)),
		VisualInfos:    make([]os.FileInfo, len(dirEntries)),
	}
}

const (
	nameColIndex     = 0
	sizeColIndex     = 1
	modifiedColIndex = 2
)

func (r *FileRows) GetCell(row, col int) *tview.TableCell {
	if row < 0 {
		return nil
	}
	if row == 0 {
		th := func(text string) *tview.TableCell {
			return tview.NewTableCell(text)
		}
		switch col {
		case nameColIndex:
			return th(" ..").SetExpansion(1)
		case sizeColIndex:
			return th("")
		case modifiedColIndex:
			return th("")
		default:
			return nil
		}
	}
	if r.Err != nil {
		if col == nameColIndex {
			return tview.NewTableCell(" 📁" + r.Err.Error()).SetTextColor(tcell.ColorOrangeRed)
		}
		return nil
	}
	if len(r.VisibleEntries) == 0 {
		if col == nameColIndex {
			return tview.NewTableCell("[::i]No entries[::-]").SetTextColor(tcell.ColorGray)
		}
		return nil
	}
	i := row - 1
	dirEntry := r.VisibleEntries[i]
	var cell *tview.TableCell
	name := dirEntry.Name()
	if col == nameColIndex {
		if dirEntry.IsDir() {
			cell = tview.NewTableCell(" 📁" + name)
		} else {
			cell = tview.NewTableCell(" 📄" + name)
		}
	} else {
		fi := r.VisualInfos[i]
		if fi == nil {
			var err error
			fi, err = dirEntry.Info()
			if err != nil {
				return tview.NewTableCell(err.Error()).SetBackgroundColor(tcell.ColorRed)
			}
			r.Infos[i] = fi
		}

		switch col {
		case sizeColIndex:
			var sizeText string
			if !dirEntry.IsDir() {
				size := fi.Size()
				sizeText = fsutils.GetSizeShortText(size)
			}
			cell = tview.NewTableCell(sizeText).
				SetAlign(tview.AlignRight).
				SetExpansion(1)
		case modifiedColIndex:
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
	cell.SetReference(fsutils.ExpandHome(path.Join(r.NodePath, name)))
	return cell
}
