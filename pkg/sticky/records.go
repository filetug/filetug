package sticky

import "github.com/rivo/tview"

type Records interface {
	Count() int
	GetCell(row, col int, name string) *tview.TableCell
}
