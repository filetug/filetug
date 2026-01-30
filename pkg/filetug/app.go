package filetug

import "github.com/rivo/tview"

type ftApp struct {
	*tview.Application
}

func (a ftApp) QueueUpdateDraw(f func()) {
	_ = a.Application.QueueUpdateDraw(f)
}

func (a ftApp) SetFocus(p tview.Primitive) {
	_ = a.Application.SetFocus(p)
}
