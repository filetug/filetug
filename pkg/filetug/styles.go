package filetug

import (
	"github.com/gdamore/tcell/v2"
)

type Theme struct {
	FocusedBorderColor       tcell.Color
	FocusedGraphicsColor     tcell.Color
	FocusedSelectedTextStyle tcell.Style

	BlurredBorderColor       tcell.Color
	BlurredGraphicsColor     tcell.Color
	BlurredSelectedTextStyle tcell.Style

	TableHeaderColor tcell.Color

	HotkeyColor tcell.Color
}

var theme = Theme{
	FocusedBorderColor:   tcell.ColorCornflowerBlue,
	FocusedGraphicsColor: tcell.ColorWhite,
	FocusedSelectedTextStyle: tcell.StyleDefault.
		Background(tcell.ColorWhite).
		Foreground(tcell.ColorBlack),

	BlurredBorderColor:   tcell.ColorGray,
	BlurredGraphicsColor: tcell.ColorGray,
	BlurredSelectedTextStyle: tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorWhite),

	TableHeaderColor: tcell.ColorWhiteSmoke,
	HotkeyColor:      tcell.ColorWhite,
}
