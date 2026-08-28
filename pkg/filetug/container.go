package filetug

import "github.com/rivo/tview"

type Container struct {
	*tview.Flex
	index     int
	content   tview.Primitive
	secondary tview.Primitive
	nav       *Navigator
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
	r.render()
}

func (r *Container) SetSecondary(p tview.Primitive) {
	if r == nil || r.Flex == nil {
		return
	}
	r.secondary = p
	r.render()
}

func (r *Container) render() {
	r.Clear()
	if r.content != nil {
		r.AddItem(r.content, 0, 1, false)
	}
	if r.secondary != nil {
		r.SetDirection(tview.FlexRow)
		r.AddItem(r.secondary, 0, 1, false)
	}
}
