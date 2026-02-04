package filetug

import "github.com/rivo/tview"

type Container struct {
	*tview.Flex
	index   int
	content tview.Primitive
	nav     *Navigator
}

func NewContainer(index int, nav *Navigator) *Container {
	r := &Container{
		Flex:  tview.NewFlex(),
		index: index,
		nav:   nav,
	}
	r.SetFocusFunc(func() {
		if r.content == nil {
			r.nav.app.SetFocus(r)
		} else {
			r.nav.app.SetFocus(r.content)
		}
	})
	return r
}

func (r *Container) SetContent(p tview.Primitive) {
	if r == nil || r.Flex == nil {
		return
	}
	r.content = p
	r.Clear()
	r.AddItem(p, 0, 1, false)
}
