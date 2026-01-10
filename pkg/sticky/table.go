package sticky

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Table struct {
	*tview.Table
	width   int
	columns []Column
	records Records
	//
	topRowIndex int
}

func (t *Table) SetRecords(records Records) {
	t.records = records
	t.render()
}

func NewTable(columns []Column) *Table {
	t := &Table{
		columns: columns,
		Table:   tview.NewTable(),
	}
	// ---- re-render on resize ----
	//app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
	//	render()
	//	return false
	//})
	t.setHeader()
	t.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		t.width = width
		return x + 1, y + 1, width - 2, height - 2
	})
	// ---- keyboard scrolling ----
	t.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			if t.topRowIndex < t.records.Count()-1 {
				t.topRowIndex++
				t.render()
			}
			return nil
		case tcell.KeyUp:
			if t.topRowIndex > 0 {
				t.topRowIndex--
				t.render()
			}
			return nil
		default:
			return event
		}
	})
	return t
}

func (t *Table) setHeader() {
	for i, col := range t.columns {
		t.SetCell(0, i, tview.NewTableCell(col.Name))
	}
}

func (t *Table) render() {
	t.Clear()
	t.setHeader()

	_, _, _, visibleRowsCount := t.GetRect()

	if visibleRowsCount <= 0 {
		return
	}

	remainingWidth := t.width

	maxColWidth := make([]int, len(t.columns))
	{ // We should do this initially and on resize, not for each render
		for i, col := range t.columns {
			if col.FixedWidth > 0 {
				maxColWidth[i] = col.FixedWidth
				remainingWidth += col.FixedWidth
				continue
			}
			for _, column := range t.columns[i+1:] {
				if column.FixedWidth > 0 {
					maxColWidth[i] -= column.FixedWidth
				}
			}
			if maxColWidth[i] > col.FixedWidth {
				maxColWidth[i] = col.FixedWidth
			}
			if col.MinWidth > 0 && maxColWidth[i] < col.MinWidth {
				maxColWidth[i] = col.MinWidth
			}
			remainingWidth -= maxColWidth[i]
		}
	}

	for row := t.topRowIndex; row < visibleRowsCount && t.topRowIndex+row < t.records.Count(); row++ {
		for col, column := range t.columns {
			td := t.records.GetCell(row, col, column.Name)
			if maxWidth := maxColWidth[col]; maxWidth > 0 {
				td.SetMaxWidth(maxWidth)
			}
			if column.Expansion > 0 {
				td.SetExpansion(column.Expansion)
			}
			t.SetCell(row+1, col, td)
		}
	}
}
