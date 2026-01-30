package sneatv

import "github.com/rivo/tview"

type TabsApp interface {
	QueueUpdateDraw(f func())
	SetFocus(p tview.Primitive)
}
