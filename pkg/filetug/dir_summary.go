package filetug

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type dirSummary struct {
	*sneatv.Boxed
	flex     *tview.Flex
	nav      *Navigator
	extTable *tview.Table

	dir *DirContext

	extByID  map[string]*extStat
	extStats []*extStat

	extGroupsByID map[string]*extensionsGroup
	extGroups     []*extensionsGroup
}

func newDirSummary(nav *Navigator) *dirSummary {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.SetTitle("Dir Summary")

	tabs := sneatv.NewTabs(nav.app, sneatv.UnderlineTabsStyle)
	flex.AddItem(tabs, 0, 1, false)

	extTable := tview.NewTable()
	extTable.SetSelectable(true, false)
	d := &dirSummary{
		nav: nav,
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(0, -1),
		),
		flex:     flex,
		extTable: extTable,
	}
	gitTextView := tview.NewTextView()
	gitTextView.SetText("git status info")
	tabs.AddTabs(
		sneatv.NewTab("file_types", "File types", false, d.extTable),
		sneatv.NewTab("git", "Git", false, gitTextView),
	)

	selectedStyle := tcell.StyleDefault
	selectedStyle = selectedStyle.Foreground(tcell.ColorBlack)
	selectedStyle = selectedStyle.Background(tcell.ColorWhiteSmoke)
	d.extTable.SetSelectedStyle(selectedStyle)
	//rows.AddItem(tview.NewTextView().SetText("By extension").SetTextColor(tcell.ColorDarkGray), 1, 0, false)
	//flex.AddItem(d.extTable, 0, 1, false)

	d.extTable.SetInputCapture(d.inputCapture)
	d.extTable.SetSelectionChangedFunc(d.selectionChanged)

	return d
}

type extensionsGroup struct {
	id    string
	title string
	*groupStats
	extStats []*extStat
}

type groupStats struct {
	Count     int
	TotalSize int64
}

type extStat struct {
	id string
	groupStats
	entries []os.DirEntry
}

func (d *dirSummary) Focus(delegate func(p tview.Primitive)) {
	//if row, _ := d.extTable.GetSelection(); row < 0 && d.extTable.GetRowCount() > 0 {
	//	d.extTable.Select(0, 0)
	//}
	d.extTable.Focus(delegate)
}

func (d *dirSummary) SetDir(dir *DirContext) {
	d.dir = dir

	d.extByID = make(map[string]*extStat)
	d.extStats = make([]*extStat, 0)
	d.extGroupsByID = make(map[string]*extensionsGroup)
	d.extGroups = make([]*extensionsGroup, 0)
	for _, entry := range dir.children {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		extID := path.Ext(name)
		if extID == name {
			continue
		}
		ext, ok := d.extByID[extID]
		if !ok {
			ext = &extStat{
				id: extID,
			}
			d.extByID[extID] = ext
			d.extStats = append(d.extStats, ext)
		}
		ext.entries = append(ext.entries, entry)
		ext.Count++

		groupID := fileExtTypes[extID]
		if groupID == "" {
			groupID = otherExtensionsGroupID
		}
		extGroup, existingExtGroup := d.extGroupsByID[groupID]

		if !existingExtGroup {
			extGroup = &extensionsGroup{
				id:         groupID,
				title:      fileExtPlurals[groupID],
				groupStats: new(groupStats),
			}
			if extGroup.title == "" {
				extGroup.title = groupID + "s"
			}
			d.extGroupsByID[groupID] = extGroup
			d.extGroups = append(d.extGroups, extGroup)
		}
		extGroup.Count++

		groupHasExt := false
		for _, extStat := range extGroup.extStats {
			if extStat.id == extID {
				groupHasExt = true
				break
			}
		}
		if !groupHasExt {
			extGroup.extStats = append(extGroup.extStats, ext)
		}
	}

	slices.SortFunc(d.extStats, func(a, b *extStat) int {
		return strings.Compare(a.id, b.id)
	})

	slices.SortFunc(d.extGroups, func(a, b *extensionsGroup) int {
		if a.id == otherExtensionsGroupID {
			return 1
		}
		if b.id == otherExtensionsGroupID {
			return -1
		}
		return strings.Compare(a.title, b.title)
	})

	for _, group := range d.extGroups {
		slices.SortFunc(group.extStats, func(a, b *extStat) int {
			return strings.Compare(a.id, b.id)
		})
	}

	d.updateTable()

	go func() {
		if err := d.GetSizes(); err == nil {
			d.nav.queueUpdateDraw(func() {
				d.updateTable()
			})
		}
	}()
}

func (d *dirSummary) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyLeft:
		d.nav.setAppFocus(d.nav.files)
		return nil
	case tcell.KeyDown:
		row, col := d.extTable.GetSelection()
		if row >= d.extTable.GetRowCount()-1 {
			return event
		}
		nextCell := d.extTable.GetCell(row+1, 1)
		switch ref := nextCell.Reference.(type) {
		case *extensionsGroup:
			if len(ref.extStats) == 1 {
				d.extTable.Select(row+2, col)
				return nil
			}
			return event
		}
		return event
	case tcell.KeyUp:
		row, col := d.extTable.GetSelection()
		if row <= 0 {
			return event
		}
		nextCell := d.extTable.GetCell(row-1, 1)
		switch ref := nextCell.Reference.(type) {
		case *extensionsGroup:
			if len(ref.extStats) == 1 {
				if row == 1 {
					return nil
				}
				d.extTable.Select(row-2, col)
				return nil
			}
			return event
		}
		return event
	default:
		return event
	}
}

func (d *dirSummary) selectionChanged(row int, _ int) {
	for i := 0; i < d.extTable.GetRowCount(); i++ {
		d.extTable.GetCell(i, 0).SetText(" ")
	}
	i := row - 1
	if row < 0 {
		return
	}

	cell1 := d.extTable.GetCell(row, 1)
	var filter ftui.Filter
	if cell1.Reference != nil {
		switch ref := cell1.Reference.(type) {
		case string:
			filter.Extensions = []string{ref}
			color := GetColorByFileExt(ref)
			cell0 := d.extTable.GetCell(i, 0)
			cell0.SetText("⇐").SetTextColor(color)
		case *extensionsGroup:
			for _, ext := range ref.extStats {
				filter.Extensions = append(filter.Extensions, ext.id)
			}
		}
	}
	d.nav.files.SetFilter(filter)
}

func (d *dirSummary) updateTable() {
	d.extTable.Clear()
	const cellTextColor = tcell.ColorLightGray

	var totalSize int64

	var row int

	for _, g := range d.extGroups {
		{
			const bgColor = 0x1a1a1a
			col := 1
			nameCell := tview.NewTableCell(" ▼ " + g.title).SetExpansion(1)
			nameCell.SetReference(g)
			nameCell.SetBackgroundColor(bgColor)
			d.extTable.SetCell(row, 1, nameCell)
			col++

			var countText string
			if len(g.extStats) > 1 {
				if g.Count == 1 {
					countText = "[ghostwhite]1[-] file "
				} else {
					countText = fmt.Sprintf("[ghostwhite]%d[-] filesPanel", g.Count)
				}
			}
			countCell := tview.NewTableCell(countText).SetAlign(tview.AlignRight).SetTextColor(cellTextColor)
			countCell.SetBackgroundColor(bgColor)
			d.extTable.SetCell(row, col, countCell)
			col++

			var sizeCell *tview.TableCell
			if len(g.extStats) > 1 {
				sizeCell = getSizeCell(g.TotalSize, cellTextColor)
			} else {
				sizeCell = tview.NewTableCell("")
			}
			sizeCell.SetBackgroundColor(bgColor)
			d.extTable.SetCell(row, col, sizeCell)

			row++
		}

		for i, ext := range g.extStats {
			var col int
			d.extTable.SetCell(i, col, tview.NewTableCell(" "))
			col++

			nameText := "  *" + ext.id
			if nameText == "*" {
				nameText = "<no extension>"
			}
			nameColor := GetColorByFileExt(nameText)
			nameCell := tview.NewTableCell(nameText)
			nameCell.SetExpansion(1)
			nameCell.SetTextColor(nameColor)
			nameCell.SetReference(ext.id)

			d.extTable.SetCell(row, col, nameCell)
			col++

			var countText string
			if ext.Count == 1 {
				countText = "[ghostwhite]1[-] file "
			} else {
				countText = fmt.Sprintf("[ghostwhite]%d[-] filesPanel", ext.Count)
			}

			countCell := tview.NewTableCell(countText).SetAlign(tview.AlignRight).SetTextColor(cellTextColor)
			d.extTable.SetCell(row, col, countCell)
			col++

			totalSize += ext.TotalSize

			sizeCell := getSizeCell(ext.TotalSize, cellTextColor)
			d.extTable.SetCell(row, col, sizeCell)
			col++

			row++
		}
	}
}

func getSizeCell(size int64, defaultColor tcell.Color) (sizeCell *tview.TableCell) {
	shortText := fsutils.GetSizeShortText(size)
	sizeText := "  " + shortText
	sizeCell = tview.NewTableCell(sizeText).SetAlign(tview.AlignRight)
	if size >= 1024*1024*1024*1024 { // TB
		sizeCell.SetTextColor(tcell.ColorOrangeRed)
	} else if size >= 1024*1024*1024 { // GB
		sizeCell.SetTextColor(tcell.ColorYellow)
	} else if size >= 1024*1024 { // MB
		sizeCell.SetTextColor(tcell.ColorLightGreen)
	} else if size >= 1024 { // KB
		sizeCell.SetTextColor(tcell.ColorWhiteSmoke)
	} else if size > 0 {
		sizeCell.SetText(sizeText + " ")
		sizeCell.SetTextColor(defaultColor)
	} else {
		sizeCell.SetText(sizeText + " ")
		sizeCell.SetTextColor(tcell.ColorLightBlue)
	}
	return
}

func (d *dirSummary) GetSizes() error {
	for _, g := range d.extGroups {
		g.TotalSize = 0
		for _, ext := range g.extStats {
			ext.TotalSize = 0
			for _, entry := range ext.entries {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				if info == nil {
					continue
				}
				rv := reflect.ValueOf(info)
				if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func) && rv.IsNil() {
					continue
				}
				size := info.Size()
				ext.TotalSize += size
			}
			g.TotalSize += ext.TotalSize
		}
	}
	return nil
}
