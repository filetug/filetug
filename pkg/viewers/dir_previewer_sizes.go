package viewers

import (
	"reflect"

	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/colors"
)

// GetSizes calculates and updates sizes for all extension groups.
func (d *DirPreviewer) GetSizes() error {
	return getSizesForGroups(d.ExtGroups)
}

// getSizesForGroups calculates total sizes for groups and their extensions.
func getSizesForGroups(groups []*ExtensionsGroup) error {
	for _, g := range groups {
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

// GetSizeCell creates a table cell with size information and appropriate color.
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
		sizeCell.SetTextColor(colors.TableHeaderColor)
	} else if size > 0 {
		sizeCell.SetText(sizeText + " ")
		sizeCell.SetTextColor(defaultColor)
	} else {
		sizeCell.SetText(sizeText + " ")
		sizeCell.SetTextColor(colors.TableColumnTitle)
	}
	return sizeCell
}
