package viewers

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UpdateTable updates the table display with current extension statistics.
// Exported for tests - try to move/refactor tests and remove.
func (d *DirPreviewer) UpdateTable() {
	d.updateTable()
}

func (d *DirPreviewer) updateTable() {
	d.tableMu.Lock()
	defer d.tableMu.Unlock()
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
