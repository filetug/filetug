package filetug

import "github.com/rivo/tview"

type container struct {
	*tview.Flex
	index int
	inner tview.Primitive
	nav   *Navigator
}

func newContainer(index int, nav *Navigator) *container {
	r := &container{
		Flex:  tview.NewFlex(),
		index: index,
		nav:   nav,
	}
	r.SetFocusFunc(func() {
		if r.inner != nil {
			r.nav.setAppFocus(r.inner)
		}
	})
	return r
}

func (r *container) SetContent(p tview.Primitive) {
	r.inner = p
	r.Clear()
	r.AddItem(p, 0, 1, false)
}
