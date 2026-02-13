package viewers

import (
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/gdamore/tcell/v2"
)

// InputCapture handles keyboard input for the directory previewer.
func (d *DirPreviewer) InputCapture(event *tcell.EventKey) *tcell.EventKey {
	return d.inputCapture(event)
}

func (d *DirPreviewer) inputCapture(event *tcell.EventKey) *tcell.EventKey {
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

func (d *DirPreviewer) selectionChanged(row int, _ int) {
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
			cell0 := d.ExtTable.GetCell(i+1, 0)
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
