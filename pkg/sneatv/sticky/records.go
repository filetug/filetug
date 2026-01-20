package sticky

import "github.com/rivo/tview"

type Records interface {
	RecordsCount() int
	GetCell(row, col int) *tview.TableCell
}
