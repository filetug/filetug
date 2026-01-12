package filetug

import "github.com/rivo/tview"

type right struct {
	inner tview.Primitive
	*tview.Flex
	nav *Navigator
}

func newRight(nav *Navigator) *right {
	r := &right{
		Flex: tview.NewFlex(),
		nav:  nav,
	}
	r.SetFocusFunc(func() {
		if r.inner != nil {
			r.nav.app.SetFocus(r.inner)
		}
	})
	return r
}

func (r *right) SetContent(p tview.Primitive) {
	r.inner = p
	r.Clear()
	r.AddItem(p, 0, 1, false)
}
