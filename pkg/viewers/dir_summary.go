package viewers

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DirSummaryOption func(*DirSummary)

func WithDirSummaryFilterSetter(setter func(ftui.Filter)) DirSummaryOption {
	return func(d *DirSummary) {
		d.setFilter = setter
	}
}

func WithDirSummaryFocusLeft(setter func()) DirSummaryOption {
	return func(d *DirSummary) {
		d.focusLeft = setter
	}
}

func WithDirSummaryQueueUpdateDraw(setter func(func())) DirSummaryOption {
	return func(d *DirSummary) {
		d.queueUpdateDraw = setter
	}
}

func WithDirSummaryColorByExt(setter func(string) tcell.Color) DirSummaryOption {
	return func(d *DirSummary) {
		d.colorByExt = setter
	}
}

type DirSummary struct {
	*sneatv.Boxed
	flex *tview.Flex
	tabs *sneatv.Tabs

	ExtTable     *tview.Table
	GitPreviewer *GitDirStatusPreviewer

	app *tview.Application

	dirPath string

	extByID  map[string]*ExtStat
	ExtStats []*ExtStat

	extGroupsByID map[string]*ExtensionsGroup
	ExtGroups     []*ExtensionsGroup

	setFilter       func(ftui.Filter)
	focusLeft       func()
	queueUpdateDraw func(func())
	colorByExt      func(string) tcell.Color
}

func NewDirSummary(app *tview.Application, options ...DirSummaryOption) *DirSummary {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.SetTitle("Dir Summary")

	extTable := tview.NewTable()
	extTable.SetSelectable(true, false)

	d := &DirSummary{
		app:      app,
		flex:     flex,
		ExtTable: extTable,
	}
	d.Boxed = sneatv.NewBoxed(
		flex,
		sneatv.WithLeftBorder(0, -1),
	)
	d.colorByExt = func(_ string) tcell.Color { return tcell.ColorWhiteSmoke }
	d.GitPreviewer = NewGitDirStatusPreviewer()
	d.setTabs(false)

	selectedStyle := tcell.StyleDefault
	selectedStyle = selectedStyle.Foreground(tcell.ColorBlack)
	selectedStyle = selectedStyle.Background(tcell.ColorWhiteSmoke)
	d.ExtTable.SetSelectedStyle(selectedStyle)

	d.ExtTable.SetInputCapture(d.inputCapture)
	d.ExtTable.SetSelectionChangedFunc(d.selectionChanged)

	for _, option := range options {
		option(d)
	}

	return d
}

func (d *DirSummary) Preview(entry files.EntryWithDirPath, _ []byte, queueUpdateDraw func(func())) {
	d.queueUpdateDraw = queueUpdateDraw
	dirPath := entry.Dir
	if entry.IsDir() {
		dirPath = entry.FullName()
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		d.SetDir(dirPath, nil)
		return
	}
	d.SetDir(dirPath, entries)
}

func (d *DirSummary) Main() tview.Primitive {
	return d
}

func (d *DirSummary) Meta() tview.Primitive {
	return nil
}

func (d *DirSummary) Focus(delegate func(p tview.Primitive)) {
	d.ExtTable.Focus(delegate)
}

func (d *DirSummary) UpdateTable() {
	d.updateTable()
}

func (d *DirSummary) InputCapture(event *tcell.EventKey) *tcell.EventKey {
	return d.inputCapture(event)
}

type ExtensionsGroup struct {
	ID    string
	Title string
	*GroupStats
	ExtStats []*ExtStat
}

type GroupStats struct {
	Count     int
	TotalSize int64
}

type ExtStat struct {
	ID string
	GroupStats
	entries []os.DirEntry
}

func (d *DirSummary) SetDir(dirPath string, entries []os.DirEntry) {
	d.dirPath = dirPath

	d.extByID = make(map[string]*ExtStat)
	d.ExtStats = make([]*ExtStat, 0)
	d.extGroupsByID = make(map[string]*ExtensionsGroup)
	d.ExtGroups = make([]*ExtensionsGroup, 0)
	for _, entry := range entries {
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
			ext = &ExtStat{
				ID: extID,
			}
			d.extByID[extID] = ext
			d.ExtStats = append(d.ExtStats, ext)
		}
		ext.entries = append(ext.entries, entry)
		ext.Count++

		groupID := fileExtTypes[extID]
		if groupID == "" {
			groupID = otherExtensionsGroupID
		}
		extGroup, existingExtGroup := d.extGroupsByID[groupID]

		if !existingExtGroup {
			extGroup = &ExtensionsGroup{
				ID:         groupID,
				Title:      fileExtPlurals[groupID],
				GroupStats: new(GroupStats),
			}
			if extGroup.Title == "" {
				extGroup.Title = groupID + "s"
			}
			d.extGroupsByID[groupID] = extGroup
			d.ExtGroups = append(d.ExtGroups, extGroup)
		}
		extGroup.Count++

		groupHasExt := false
		for _, extStat := range extGroup.ExtStats {
			if extStat.ID == extID {
				groupHasExt = true
				break
			}
		}
		if !groupHasExt {
			extGroup.ExtStats = append(extGroup.ExtStats, ext)
		}
	}

	slices.SortFunc(d.ExtStats, func(a, b *ExtStat) int {
		return strings.Compare(a.ID, b.ID)
	})

	slices.SortFunc(d.ExtGroups, func(a, b *ExtensionsGroup) int {
		if a.ID == otherExtensionsGroupID {
			return 1
		}
		if b.ID == otherExtensionsGroupID {
			return -1
		}
		return strings.Compare(a.Title, b.Title)
	})

	for _, group := range d.ExtGroups {
		slices.SortFunc(group.ExtStats, func(a, b *ExtStat) int {
			return strings.Compare(a.ID, b.ID)
		})
	}

	d.updateTable()

	hasRepo := gitutils.GetRepositoryRoot(dirPath) != ""
	d.setTabs(hasRepo)
	if hasRepo {
		d.GitPreviewer.SetDir(dirPath, d.queueUpdateDraw)
	}

	if d.queueUpdateDraw == nil {
		err := d.GetSizes()
		if err == nil {
			d.updateTable()
		}
		return
	}

	go func() {
		err := d.GetSizes()
		if err != nil {
			return
		}
		d.queueUpdate(func() {
			d.updateTable()
		})
	}()
}

func (d *DirSummary) setTabs(hasGit bool) {
	if d.tabs != nil {
		d.flex.RemoveItem(d.tabs)
	}
	tabs := sneatv.NewTabs(d.app, sneatv.UnderlineTabsStyle)
	fileTab := sneatv.NewTab("file_types", "File types", false, d.ExtTable)
	tabs.AddTabs(fileTab)
	if hasGit {
		gitTab := sneatv.NewTab("git", "Git", false, d.GitPreviewer.Main())
		tabs.AddTabs(gitTab)
	}
	d.tabs = tabs
	d.flex.AddItem(tabs, 0, 1, false)
}

func (d *DirSummary) queueUpdate(f func()) {
	if d.queueUpdateDraw != nil {
		d.queueUpdateDraw(f)
		return
	}
	f()
}

func (d *DirSummary) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyLeft:
		if d.focusLeft != nil {
			d.focusLeft()
			return nil
		}
		return event
	case tcell.KeyDown:
		row, col := d.ExtTable.GetSelection()
		if row >= d.ExtTable.GetRowCount()-1 {
			return event
		}
		nextCell := d.ExtTable.GetCell(row+1, 1)
		switch ref := nextCell.Reference.(type) {
		case *ExtensionsGroup:
			if len(ref.ExtStats) == 1 {
				d.ExtTable.Select(row+2, col)
				return nil
			}
			return event
		}
		return event
	case tcell.KeyUp:
		row, col := d.ExtTable.GetSelection()
		if row <= 0 {
			return event
		}
		nextCell := d.ExtTable.GetCell(row-1, 1)
		switch ref := nextCell.Reference.(type) {
		case *ExtensionsGroup:
			if len(ref.ExtStats) == 1 {
				if row == 1 {
					return nil
				}
				d.ExtTable.Select(row-2, col)
				return nil
			}
			return event
		}
		return event
	default:
		return event
	}
}

func (d *DirSummary) selectionChanged(row int, _ int) {
	for i := 0; i < d.ExtTable.GetRowCount(); i++ {
		cell := d.ExtTable.GetCell(i, 0)
		cell.SetText(" ")
	}
	i := row - 1
	if row < 0 {
		return
	}

	cell1 := d.ExtTable.GetCell(row, 1)
	var filter ftui.Filter
	if cell1.Reference != nil {
		switch ref := cell1.Reference.(type) {
		case string:
			filter.Extensions = []string{ref}
			cell0 := d.ExtTable.GetCell(i, 0)
			color := d.colorByExt(ref)
			cell0.SetText("⇐").SetTextColor(color)
		case *ExtensionsGroup:
			for _, ext := range ref.ExtStats {
				filter.Extensions = append(filter.Extensions, ext.ID)
			}
		}
	}
	if d.setFilter == nil {
		return
	}
	d.setFilter(filter)
}

func (d *DirSummary) updateTable() {
	d.ExtTable.Clear()
	const cellTextColor = tcell.ColorLightGray

	var row int

	for _, g := range d.ExtGroups {
		const bgColor = 0x1a1a1a
		col := 1
		nameCell := tview.NewTableCell(" ▼ " + g.Title)
		nameCell.SetExpansion(1)
		nameCell.SetReference(g)
		nameCell.SetBackgroundColor(bgColor)
		d.ExtTable.SetCell(row, col, nameCell)
		col++

		var countText string
		if len(g.ExtStats) > 1 {
			if g.Count == 1 {
				countText = "[ghostwhite]1[-] file "
			} else {
				countText = fmt.Sprintf("[ghostwhite]%d[-] filesPanel", g.Count)
			}
		}
		countCell := tview.NewTableCell(countText)
		countCell.SetAlign(tview.AlignRight)
		countCell.SetTextColor(cellTextColor)
		countCell.SetBackgroundColor(bgColor)
		d.ExtTable.SetCell(row, col, countCell)
		col++

		var sizeCell *tview.TableCell
		if len(g.ExtStats) > 1 {
			sizeCell = GetSizeCell(g.TotalSize, cellTextColor)
		} else {
			sizeCell = tview.NewTableCell("")
		}
		sizeCell.SetBackgroundColor(bgColor)
		d.ExtTable.SetCell(row, col, sizeCell)

		row++

		for _, ext := range g.ExtStats {
			col = 0
			emptyCell := tview.NewTableCell(" ")
			d.ExtTable.SetCell(row, col, emptyCell)
			col++

			nameText := "  *" + ext.ID
			if ext.ID == "" {
				nameText = "  <no extension>"
			}
			nameColor := d.colorByExt(nameText)
			nameCell := tview.NewTableCell(nameText)
			nameCell.SetExpansion(1)
			nameCell.SetTextColor(nameColor)
			nameCell.SetReference(ext.ID)
			d.ExtTable.SetCell(row, col, nameCell)
			col++

			var countText string
			if ext.Count == 1 {
				countText = "[ghostwhite]1[-] file "
			} else {
				countText = fmt.Sprintf("[ghostwhite]%d[-] filesPanel", ext.Count)
			}

			countCell := tview.NewTableCell(countText)
			countCell.SetAlign(tview.AlignRight)
			countCell.SetTextColor(cellTextColor)
			d.ExtTable.SetCell(row, col, countCell)
			col++

			sizeCell = GetSizeCell(ext.TotalSize, cellTextColor)
			d.ExtTable.SetCell(row, col, sizeCell)

			row++
		}
	}
}

func GetSizeCell(size int64, defaultColor tcell.Color) *tview.TableCell {
	shortText := fsutils.GetSizeShortText(size)
	sizeText := "  " + shortText
	sizeCell := tview.NewTableCell(sizeText)
	sizeCell.SetAlign(tview.AlignRight)
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
	return sizeCell
}

func (d *DirSummary) GetSizes() error {
	for _, g := range d.ExtGroups {
		g.TotalSize = 0
		for _, ext := range g.ExtStats {
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

var fileExtTypes = map[string]string{
	// Image file extStats
	".jpg":  "Image",
	".jpeg": "Image",
	".png":  "Image",
	".gif":  "Image",
	".bmp":  "Image",
	".riff": "Image",
	".tiff": "Image",
	".vp8":  "Image",
	".vp8l": "Image",
	".webp": "Image",

	// Video file extStats
	".mov":  "Video",
	".mp4":  "Video",
	".webm": "Video",
	// Code file extStats
	".go":   "Code",
	".css":  "Code",
	".js":   "Code",
	".cpp":  "Code",
	".java": "Code",
	".cs":   "Code",
	// Data file extStats
	".json": "Data",
	".xml":  "Data",
	".dbf":  "Data",
	// Text file extStats
	".txt": "Text",
	".md":  "Text",
	// Log file extStats
	".log": "Log",
}

const otherExtensionsGroupID = "Other"

var fileExtPlurals = map[string]string{
	"Data": "Data",
	"Code": "Code",
}
