package filetug

import (
	"github.com/gdamore/tcell/v2"
)

type Styles struct {
	FocusedBorderColor   tcell.Color
	FocusedGraphicsColor tcell.Color

	BlurBorderColor   tcell.Color
	BlurGraphicsColor tcell.Color

	TableHeaderColor tcell.Color
}

var Style = Styles{
	FocusedBorderColor:   tcell.ColorCornflowerBlue,
	FocusedGraphicsColor: tcell.ColorWhite,

	BlurBorderColor:   tcell.ColorGray,
	BlurGraphicsColor: tcell.ColorGray,

	TableHeaderColor: tcell.ColorWhiteSmoke,
}
