package sticky

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type mockRecords struct {
	count int
}

func (m *mockRecords) RecordsCount() int {
	return m.count
}

func (m *mockRecords) GetCell(row, col int) *tview.TableCell {
	return tview.NewTableCell(fmt.Sprintf("R%dC%d", row, col))
}

func TestNewTable(t *testing.T) {
	columns := []Column{
		{Name: "Col1", Expansion: 1},
		{Name: "Col2", FixedWidth: 10},
	}
	table := NewTable(columns)
	assert.NotNil(t, table)
	assert.Equal(t, 2, table.GetColumnCount())

	// Check header cells
	cell0 := table.GetCell(0, 0)
	assert.Equal(t, "Col1", cell0.Text)
	cell1 := table.GetCell(0, 1)
	assert.Equal(t, "Col2", cell1.Text)
}

func TestTable_SetRecords(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	records := &mockRecords{count: 5}

	// We need to set a size for render to do something
	table.SetRect(0, 0, 100, 10)
	// Sticky table uses t.width which is set in DrawFunc
	// But it is also used in render() which is called by SetRecords.
	// In the current implementation, t.width might be 0 if Draw hasn't happened.

	table.SetRecords(records)

	// After SetRecords, render is called.
	// Since visibleRowsCount from GetRect (10) is > 0, it should render some rows.
	// Header is at row 0, records start at row 1.
	assert.Equal(t, 6, table.GetRowCount()) // 1 header + 5 records
	assert.Equal(t, "R0C0", table.GetCell(1, 0).Text)
}

func TestTable_ScrollToRow(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	records := &mockRecords{count: 100}
	table.SetRect(0, 0, 100, 10) // 10 rows total, 1 header -> 9 visible records
	table.SetRecords(records)

	// Initial state
	assert.Equal(t, 0, table.topRowIndex)

	// Scroll to row 20
	table.ScrollToRow(20)
	// topRowIndex should be 20 - 9 + 1 = 12
	assert.Equal(t, 12, table.topRowIndex)

	// Scroll back to row 5
	table.ScrollToRow(5)
	assert.Equal(t, 5, table.topRowIndex)

	// Scroll to row 10 (already visible since top=5, visible=9 -> 5..13)
	table.ScrollToRow(10)
	assert.Equal(t, 5, table.topRowIndex)
}

func TestTable_InputCapture(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	records := &mockRecords{count: 100}
	table.SetRect(0, 0, 100, 10)
	table.SetRecords(records)

	inputCapture := table.GetInputCapture()
	assert.NotNil(t, inputCapture)

	// Test KeyDown
	eventDown := tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone)
	inputCapture(eventDown)
	assert.Equal(t, 1, table.topRowIndex)

	// Test KeyUp
	eventUp := tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone)
	inputCapture(eventUp)
	assert.Equal(t, 0, table.topRowIndex)
}

func TestTable_Select(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	records := &mockRecords{count: 100}
	table.SetRect(0, 0, 100, 10)
	table.SetRecords(records)

	table.Select(20, 0)
	// Selecting row 20 (record 19) should trigger ScrollToRow(19)
	// topRowIndex should be 19 - 9 + 1 = 11
	assert.Equal(t, 11, table.topRowIndex)
}

func TestTable_DrawFunc(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)

	// DrawFunc is set in NewTable
	// It should update table.width

	// We can't easily call the DrawFunc directly because it's anonymous and not exported.
	// But we know it's set via t.SetDrawFunc.
	// tview.Box (which tview.Table embeds) has Draw function which calls the drawFunc.

	screen := tcell.NewSimulationScreen("")
	table.Draw(screen) // This should trigger the DrawFunc

	// The DrawFunc in NewTable is:
	// t.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	// 	t.width = width
	// 	return x + 1, y + 1, width - 2, height - 2
	// })

	_, _, width, _ := table.GetRect()
	assert.Equal(t, width, table.width)
}

func TestTable_ScrollToRow_EdgeCases(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)

	// Case: visibleRowsCount <= 1
	table.SetRect(0, 0, 100, 1)
	table.ScrollToRow(5)
	assert.Equal(t, 0, table.topRowIndex)

	// Case: row >= t.topRowIndex+visibleRowsCount and t.topRowIndex < 0
	// visibleRowsCount = 10, header = 1, records = 9
	table.SetRect(0, 0, 100, 11) // visibleRowsCount = 11, record rows = 10
	table.topRowIndex = 0
	// row = 5. visibleRowsCount - 1 = 10.
	// if 5 >= 0 + 10 (false)

	// To trigger t.topRowIndex < 0:
	// row < visibleRowsCount - 1
	// Let's say row = 2, visibleRowsCount = 10 (9 records)
	// 2 >= 0 + 9 (false)

	// If row = 10, visibleRowsCount = 5 (4 records)
	table.SetRect(0, 0, 100, 5) // visibleRowsCount = 4 records
	table.ScrollToRow(10)
	// 10 >= 0 + 4 (true)
	// topRowIndex = 10 - 4 + 1 = 7.
	assert.Equal(t, 7, table.topRowIndex)

	// To get topRowIndex < 0:
	// row - visibleRowsCount + 1 < 0
	// row=1, visible=5. 1 - 4 + 1 = -2.
	table.topRowIndex = 5
	table.ScrollToRow(1)
	// 1 < 5 (true) -> topRowIndex = 1.
	assert.Equal(t, 1, table.topRowIndex)

	// Wait, the logic is:
	/*
		if row < t.topRowIndex {
			t.topRowIndex = row
			t.render()
		} else if row >= t.topRowIndex+visibleRowsCount {
			t.topRowIndex = row - visibleRowsCount + 1
			if t.topRowIndex < 0 {
				t.topRowIndex = 0
			}
			t.render()
		}
	*/
	// To hit the second branch AND topRowIndex < 0:
	// t.topRowIndex = 0
	// row >= visibleRowsCount
	// AND row - visibleRowsCount + 1 < 0  => row < visibleRowsCount - 1
	// This is a contradiction: row >= visibleRowsCount AND row < visibleRowsCount - 1.

	// Ah! visibleRowsCount in ScrollToRow is:
	// _, _, _, visibleRowsCount := t.GetRect()
	// visibleRowsCount-- // header

	// So if GetRect height is 5, visibleRowsCount becomes 4.
	// row >= 0 + 4 (row >= 4)
	// topRowIndex = row - 4 + 1 = row - 3.
	// If row=4, topRowIndex = 1.
	// If row=2, it goes to FIRST branch (row < topRowIndex) if topRowIndex > 2.

	// The only way to hit topRowIndex < 0 is if visibleRowsCount is large and row is small,
	// but then it would likely hit the first branch.
}

func TestTable_Render_EdgeCases(t *testing.T) {
	// Case: visibleRowsCount <= 0
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	table.SetRect(0, 0, 100, 0)
	table.render() // Should return early

	// Case: GetCell returns nil
	records := &mockNilRecords{count: 5}
	table.SetRect(0, 0, 100, 10)
	table.SetRecords(records)
	assert.Equal(t, 6, table.GetRowCount())
}

type mockNilRecords struct {
	count int
}

func (m *mockNilRecords) RecordsCount() int {
	return m.count
}

func (m *mockNilRecords) GetCell(row, col int) *tview.TableCell {
	_, _ = row, col
	return nil
}

func TestTable_Render_ColumnWidths(t *testing.T) {
	columns := []Column{
		{Name: "Fixed", FixedWidth: 10},
		{Name: "Min", MinWidth: 5},
		{Name: "Max", FixedWidth: 20}, // Will be used in maxColWidth[i] = col.FixedWidth
		{Name: "Exp", Expansion: 1},
	}
	table := NewTable(columns)
	table.width = 100
	table.SetRect(0, 0, 100, 10)
	records := &mockRecords{count: 5}
	table.SetRecords(records)

	// Just verify it doesn't crash and renders something
	assert.Equal(t, 6, table.GetRowCount())
}

func TestTable_InputCapture_Boundaries(t *testing.T) {
	columns := []Column{{Name: "Col1"}}
	table := NewTable(columns)
	records := &mockRecords{count: 10}
	table.SetRect(0, 0, 100, 5) // 4 visible records
	table.SetRecords(records)

	// table.GetRowCount() should be 1 + 5 = 6 (header + visible records)
	// Actually, visibleRowsCount=5. row < 5 and topRowIndex+row < records.RecordsCount().
	// row=0, 1, 2, 3, 4.
	// SetCell(row+1, col, td) -> rows 1, 2, 3, 4, 5. Plus header at 0. Total 6 rows.
	assert.Equal(t, 6, table.GetRowCount())

	inputCapture := table.GetInputCapture()

	// topRowIndex is 0
	// KeyUp at top
	eventUp := tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone)
	inputCapture(eventUp)
	assert.Equal(t, 0, table.topRowIndex)

	// KeyDown
	eventDown := tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone)
	inputCapture(eventDown) // topRowIndex -> 1
	assert.Equal(t, 1, table.topRowIndex)

	// KeyDown until we can't anymore
	// t.GetRowCount()-1 = 5.
	inputCapture(eventDown) // 2
	inputCapture(eventDown) // 3
	inputCapture(eventDown) // 4
	assert.Equal(t, 4, table.topRowIndex)
	inputCapture(eventDown) // 5
	assert.Equal(t, 5, table.topRowIndex)

	inputCapture(eventDown) // should stay 5 because 5 < 5 is false
	assert.Equal(t, 5, table.topRowIndex)

	// Default case
	eventLeft := tcell.NewEventKey(tcell.KeyLeft, ' ', tcell.ModNone)
	res := inputCapture(eventLeft)
	assert.Equal(t, eventLeft, res)
}
