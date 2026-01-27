package filetug

import (
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//var _ sticky.Records = (*FileRows)(nil)

var _ tview.TableContent = (*FileRows)(nil)

func NewFileRows(dir *DirContext) *FileRows {
	if dir.Path != "/" {
		dir.Path = strings.TrimSuffix(dir.Path, "/")
	}
	entries := dir.Entries()
	return &FileRows{
		store:          dir.Store,
		Dir:            dir,
		AllEntries:     entries,
		VisibleEntries: entries,
		Infos:          make([]os.FileInfo, len(entries)),
		VisualInfos:    make([]os.FileInfo, len(entries)),
	}
}

type FileRows struct {
	tview.TableContentReadOnly
	hideParent     bool
	store          files.Store
	Dir            *DirContext
	AllEntries     []files.EntryWithDirPath
	VisibleEntries []files.EntryWithDirPath
	Infos          []os.FileInfo
	VisualInfos    []os.FileInfo
	Err            error
	filter         ftui.Filter
}

func (r *FileRows) HideParent() bool {
	return r.hideParent || r.Dir.Path == "/"
}

//func (r *FileRows) SetSelected(row int) {
//	if row == 0 {
//		r.selected = ""
//	}
//	r.selected = r.AllEntries[row-1].Name()
//}

func (r *FileRows) SetFilter(filter ftui.Filter) {
	r.filter = filter
	r.applyFilter()
}

func (r *FileRows) applyFilter() {
	r.VisibleEntries = make([]files.EntryWithDirPath, 0, len(r.AllEntries))
	r.VisualInfos = make([]os.FileInfo, 0, len(r.VisibleEntries))
	for i, entry := range r.AllEntries {
		if r.filter.IsVisible(entry) {
			r.VisibleEntries = append(r.VisibleEntries, entry)
			r.VisualInfos = append(r.VisualInfos, r.Infos[i])
		}
	}
}

func (r *FileRows) GetRowCount() int {
	if r.HideParent() {
		return len(r.VisualInfos)
	}
	return len(r.VisibleEntries) + 1
}

func (r *FileRows) GetColumnCount() int {
	return 3
}

const (
	nameColIndex     = 0
	sizeColIndex     = 1
	modifiedColIndex = 2
)

func (r *FileRows) GetCell(row, col int) *tview.TableCell {
	if !r.HideParent() && row == 0 {
		return r.getTopRow(col)
	}
	if r.Err != nil {
		if col == nameColIndex {
			return tview.NewTableCell(" 📁" + r.Err.Error()).SetTextColor(tcell.ColorOrangeRed)
		}
		return nil
	}
	i := row
	if !r.HideParent() {
		i--
	}
	if i < 0 {
		return nil
	}
	if i >= len(r.VisibleEntries) && len(r.VisibleEntries) > 0 {
		return nil
	}
	var cell *tview.TableCell
	if len(r.VisibleEntries) == 0 {
		if col == nameColIndex {
			cell = tview.NewTableCell("[::i]No entries[::-]").SetTextColor(tcell.ColorGray)
		} else {
			return nil
		}
	} else {
		dirEntry := r.VisibleEntries[i]

		name := dirEntry.Name()
		if col == nameColIndex {
			isDir := dirEntry.IsDir()
			if !isDir {
				isDir = r.isSymlinkToDir(dirEntry)
			}
			if isDir {
				cell = tview.NewTableCell(dirEmoji + name)
			} else {
				cell = tview.NewTableCell("📄" + name)
			}
		} else {
			fi := r.VisualInfos[i]
			if fi == nil || reflect.ValueOf(fi).IsNil() {
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
					if fi != nil && !reflect.ValueOf(fi).IsNil() {
						size := fi.Size()
						sizeText = fsutils.GetSizeShortText(size)
					}
				}
				cell = tview.NewTableCell(sizeText).
					SetAlign(tview.AlignRight).
					SetExpansion(1)
			case modifiedColIndex:
				var s string
				if fi != nil && !reflect.ValueOf(fi).IsNil() {
					if modTime := fi.ModTime(); fi.ModTime().After(time.Now().Add(24 * time.Hour)) {
						s = modTime.Format("15:04:05")
					} else {
						s = modTime.Format("2006-01-02")
					}
				}
				cell = tview.NewTableCell(s)
			default:
				return nil
			}
		}
		color := GetColorByFileExt(name)
		cell.SetTextColor(color)
		ref := files.NewEntryWithDirPath(dirEntry, path.Join(fsutils.ExpandHome(r.Dir.Path)))
		cell.SetReference(ref)
	}
	return cell
}

func (r *FileRows) isSymlinkToDir(entry files.EntryWithDirPath) bool {
	if entry.Type()&os.ModeSymlink == 0 {
		return false
	}
	if r.store == nil || r.store.RootURL().Scheme != "file" {
		return false // TODO: Handle for FTP, HTTP & other stores?
	}
	fullName := entry.FullName()
	info, err := os.Stat(fullName) // We can't use entry.Info() as it would return symlink info
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (r *FileRows) getTopRow(col int) *tview.TableCell {
	th := func(text string) *tview.TableCell {
		return tview.NewTableCell(text)
	}
	switch col {
	case nameColIndex:
		return r.getTopRowName()
	case sizeColIndex:
		return th("")
	case modifiedColIndex:
		return th("")
	default:
		return nil
	}
}

func (r *FileRows) getTopRowName() *tview.TableCell {
	var cellText string
	rootPath := r.store.RootURL().Path
	if r.Dir.Path == rootPath {
		cellText = "."
	} else {
		cellText = ".."
	}
	cell := tview.NewTableCell(cellText).SetExpansion(1)
	var parentDir string
	if r.Dir.Path == "~" {
		parentDir = fsutils.ExpandHome("~")
	} else {
		parentDir, _ = path.Split(r.Dir.Path)
	}
	if parentDir != "/" {
		parentDir = strings.TrimSuffix(parentDir, "/")
	}
	parentDirName, parentDirPath := path.Split(parentDir)
	parentDirEntry := files.NewDirEntry(parentDirName, true)
	ref := files.NewEntryWithDirPath(parentDirEntry, parentDirPath)
	return cell.SetReference(ref)
}
