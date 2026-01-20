package crumbs

import "github.com/gdamore/tcell/v2"

type Breadcrumb interface {
	GetTitle() string
	SetTitle(string) Breadcrumb
	SetColor(color tcell.Color) Breadcrumb
	GetColor() tcell.Color
	Action() error
}

type breadcrumb struct {
	title  string
	color  tcell.Color
	action func() error
}

func (b *breadcrumb) GetTitle() string {
	return b.title
}

func (b *breadcrumb) GetColor() tcell.Color {
	return b.color
}

func (b *breadcrumb) SetTitle(title string) Breadcrumb {
	b.title = title
	return b
}

func (b *breadcrumb) SetColor(color tcell.Color) Breadcrumb {
	b.color = color
	return b
}

func (b *breadcrumb) Action() error {
	if b.action == nil {
		return nil
	}
	return b.action()
}

func NewBreadcrumb(title string, action func() error) Breadcrumb {
	return &breadcrumb{title: title, action: action}
}
