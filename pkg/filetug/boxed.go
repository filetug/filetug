package filetug

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	focusedStyle = tcell.StyleDefault.Foreground(tcell.ColorCornflowerBlue).Background(tcell.ColorBlack)
	blurredStyle = tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
)

type boxedContent interface {
	tview.Primitive
	GetTitle() string
	SetTitle(title string) *tview.Box
	SetBorderPadding(top, bottom, left, right int) *tview.Box
	//Draw(screen tcell.Screen)
	//HasFocus() bool
	//GetRect() (x int, y int, width int, height int)
}

type boxed struct {
	boxedContent
	o boxOptions
}

type boxOptions struct {
	leftBorder   bool
	leftPadding  int
	leftOffset   int
	rightBorder  bool
	rightPadding int
	rightOffset  int

	tabs []*Tab
}

type BoxOption func(*boxOptions)

//func WithLeftPadding(padding int) BoxOption {
//	return func(opts *boxOptions) {
//		opts.leftPadding = padding
//	}
//}

//func WithRightPadding(padding int) BoxOption {
//	return func(opts *boxOptions) {
//		opts.rightPadding = padding
//	}
//}

func WithLeftBorder(padding, offset int) BoxOption {
	return func(opts *boxOptions) {
		opts.leftBorder = true
		opts.leftPadding = padding
		opts.leftOffset = offset
	}
}

func WithRightBorder(padding, offset int) BoxOption {
	return func(opts *boxOptions) {
		opts.rightBorder = true
		opts.rightPadding = padding
		opts.rightOffset = offset
	}
}

func WithTabs(tabs ...*Tab) BoxOption {
	return func(opts *boxOptions) {
		opts.tabs = append(opts.tabs, tabs...)
	}
}

type Tab struct {
	ID      string
	Title   string
	Hotkey  rune
	Checked bool
	Action  func(tab string)
}

func newBoxed(inner boxedContent, o ...BoxOption) *boxed {
	b := boxed{
		boxedContent: inner,
	}
	for _, option := range o {
		option(&b.o)
	}
	inner.SetBorderPadding(1, 1, b.o.leftPadding, b.o.rightPadding)
	return &b
}

func (b boxed) Draw(screen tcell.Screen) {
	b.boxedContent.Draw(screen)
	b.drawBorders(screen)
}

func (b boxed) drawBorders(screen tcell.Screen) {
	x, y, width, height := b.GetRect()
	lineStyle := tcell.StyleDefault
	hasFocus := b.HasFocus()
	var topLineChar rune
	if hasFocus {
		lineStyle = focusedStyle
		// Double horizontal border
		topLineChar = '═'
	} else {
		lineStyle = blurredStyle
		topLineChar = '─'
	}

	horizontalStart := x + b.o.leftOffset
	horizontalLen := width
	horizontalLen += b.o.rightOffset - b.o.leftOffset
	if hasFocus {
		if b.o.leftBorder {
			horizontalStart += 1
			horizontalLen -= 1
		}
		if b.o.rightBorder {
			horizontalLen -= 1
		}
	}

	horizontalBorder := func(y int, title string) {
		if len(title) == 0 {
			for i := 0; i < horizontalLen; i++ {
				screen.SetContent(horizontalStart+i, y, topLineChar, nil, lineStyle)
			}
			return
		}
		titleWidth := tview.TaggedStringWidth(title)
		leftLen := (horizontalLen - titleWidth) / 2
		for i := 0; i < leftLen-1; i++ {
			screen.SetContent(horizontalStart+i, y, topLineChar, nil, lineStyle)
		}
		if hasFocus {
			screen.SetContent(horizontalStart+leftLen-1, y, '╡', nil, lineStyle)
		} else {
			screen.SetContent(horizontalStart+leftLen-1, y, '┤', nil, lineStyle)
		}

		tview.Print(screen, title, horizontalStart+leftLen, y, titleWidth, tview.AlignLeft, tcell.ColorGhostWhite)

		rightStart := horizontalStart + leftLen + titleWidth
		if hasFocus {
			screen.SetContent(rightStart, y, '╞', nil, lineStyle)
		} else {
			screen.SetContent(rightStart, y, '├', nil, lineStyle)
		}
		rightLen := horizontalLen - leftLen - titleWidth
		for i := 1; i < rightLen; i++ {
			screen.SetContent(rightStart+i, y, topLineChar, nil, lineStyle)
		}
	}

	var title string
	if len(b.o.tabs) == 0 {
		title = b.GetTitle()
	} else {
		var sb strings.Builder
		for i, tab := range b.o.tabs {
			if i > 0 {
				sb.WriteString("[gray]|[-]")
			}
			if tab.Checked {
				sb.WriteString("☑️")
			} else {
				sb.WriteString("🔲")
			}
			title = tab.Title
			if tab.Hotkey != 0 {
				title = strings.Replace(title, string(tab.Hotkey),
					fmt.Sprintf("[%s]%c[-][DarkGray]", theme.HotkeyColor, tab.Hotkey), 1)
			}
			title = fmt.Sprintf("[DarkGray]%s[-]", title)
			sb.WriteString(title)
		}
		title = sb.String()
	}

	horizontalBorder(y, title) // top line

	verticalBorder := func(x int, top, bottom rune) {
		screen.SetContent(x, y, top, nil, lineStyle)

		for i := 1; i < height-1; i++ {
			screen.SetContent(x, y+i, '│', nil, lineStyle)
		}

		screen.SetContent(x, y+height-1, bottom, nil, lineStyle)

		//if hasFocus {
		//	screen.SetContent(x, y+height-1, bottom, nil, lineStyle)
		//} else {
		//	screen.SetContent(x, y+height-1, topLineChar, nil, lineStyle)
		//}
	}

	if b.o.leftBorder {
		if hasFocus {
			verticalBorder(x+b.o.leftOffset, '╒', '╘')
		} else {
			verticalBorder(x+b.o.leftOffset, '┬', '┴')
		}
	}
	if b.o.rightBorder {
		if hasFocus {
			verticalBorder(x+width-1+b.o.rightOffset, '╕', '╛')
		} else {
			verticalBorder(x+width-1+b.o.rightOffset, '┬', '┴')
		}
	}

	horizontalBorder(y+height-1, "") // bottom line
}
