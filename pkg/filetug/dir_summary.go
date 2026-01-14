package filetug

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type dirSummary struct {
	*boxed
	flex     *tview.Flex
	nav      *Navigator
	extTable *tview.Table
	extStats map[string]*groupStats
	dir      *DirContext
}

type groupStats struct {
	Count     int
	TotalSize int64
}

type extStats struct {
	Ext string
	*groupStats
}

func (d *dirSummary) Focus(delegate func(p tview.Primitive)) {
	//if row, _ := d.extTable.GetSelection(); row < 0 && d.extTable.GetRowCount() > 0 {
	//	d.extTable.Select(0, 0)
	//}
	d.extTable.Focus(delegate)
}

func newDirSummary(dir *DirContext, nav *Navigator) *dirSummary {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetTitle("Dir Summary")
	d := &dirSummary{
		nav: nav,
		dir: dir,
		boxed: newBoxed(
			flex,
			WithLeftBorder(0, -1),
		),
		flex:     flex,
		extTable: tview.NewTable().SetSelectable(true, false),
	}
	flex.AddItem(tview.NewTextView().SetText("By extension").SetTextColor(tcell.ColorDarkGray), 1, 0, false)
	flex.AddItem(d.extTable, 0, 1, false)
	d.extStats = make(map[string]*groupStats)
	extensions := make([]extStats, 0)
	for _, entry := range dir.children {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := path.Ext(name)
		if ext == name {
			continue
		}
		stat, ok := d.extStats[ext]
		if !ok {
			stat = new(groupStats)
			d.extStats[ext] = stat
			extensions = append(extensions, extStats{Ext: ext, groupStats: stat})
		}
		stat.Count++
	}
	slices.SortFunc(extensions, func(a, b extStats) int {
		return strings.Compare(a.Ext, b.Ext)
	})

	fillTable := func() {
		d.extTable.Clear()
		const cellTextColor = tcell.ColorLightGray
		for row, ext := range extensions {
			nameText := "*" + ext.Ext
			if nameText == "*" {
				nameText = "<no extension>"
			}
			nameColor := GetColorByFileExt(nameText)
			nameCell := tview.NewTableCell(nameText)
			nameCell.SetExpansion(1)
			nameCell.SetTextColor(nameColor)
			d.extTable.SetCell(row, 0, nameCell)

			var countText string
			if ext.Count == 1 {
				countText = "[ghostwhite]1[-] file "
			} else {
				countText = fmt.Sprintf("[ghostwhite]%d[-] files", ext.Count)
			}
			countCell := tview.NewTableCell(countText).SetAlign(tview.AlignRight).SetTextColor(cellTextColor)
			d.extTable.SetCell(row, 1, countCell)

			sizeText := "  " + fsutils.GetSizeShortText(ext.TotalSize)
			sizeCell := tview.NewTableCell(sizeText).SetAlign(tview.AlignRight)
			if ext.TotalSize >= 1024*1024*1024*1024 { // TB
				sizeCell.SetTextColor(tcell.ColorOrangeRed)
			} else if ext.TotalSize >= 1024*1024*1024 { // GB
				sizeCell.SetTextColor(tcell.ColorYellow)
			} else if ext.TotalSize >= 1024*1024 { // MB
				sizeCell.SetTextColor(tcell.ColorLightGreen)
			} else if ext.TotalSize >= 1024 { // KB
				sizeCell.SetTextColor(tcell.ColorWhiteSmoke)
			} else if ext.TotalSize > 0 {
				sizeCell.SetText(sizeText + " ")
				sizeCell.SetTextColor(cellTextColor)
			} else {
				sizeCell.SetText(sizeText + " ")
				sizeCell.SetTextColor(tcell.ColorLightBlue)
			}
			d.extTable.SetCell(row, 2, sizeCell)
		}
	}

	fillTable()

	go func() {
		if err := d.GetSizes(); err == nil {
			d.nav.app.QueueUpdateDraw(func() {
				fillTable()
			})
		}
	}()

	d.extTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			d.nav.app.SetFocus(d.nav.files)
			return nil
		default:
			return event
		}
	})
	return d
}

func (d *dirSummary) GetSizes() error {
	for _, stat := range d.extStats {
		stat.Count = 0
		stat.TotalSize = 0
	}
	extensions := make([]extStats, 0)
	for _, entry := range d.dir.children {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		ext := path.Ext(entry.Name())
		stat, ok := d.extStats[ext]
		if !ok {
			stat = new(groupStats)
			d.extStats[ext] = stat
			extensions = append(extensions, extStats{Ext: ext, groupStats: stat})
		}
		stat.Count++
		stat.TotalSize += info.Size()
	}
	slices.SortFunc(extensions, func(a, b extStats) int {
		return strings.Compare(a.Ext, b.Ext)
	})
	return nil
}
