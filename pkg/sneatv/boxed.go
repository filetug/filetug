package sneatv

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/themes"
)

var (
	focusedStyle = tcell.StyleDefault.Foreground(tcell.ColorCornflowerBlue).Background(tcell.ColorBlack)
	blurredStyle = tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)
)

type BoxedContent interface {
	tview.Primitive
	GetTitle() string
	SetTitle(title string) *tview.Box
	SetBorderPadding(top, bottom, left, right int) *tview.Box
	//Draw(screen tcell.Screen)
	//HasFocus() bool
	//GetRect() (x int, y int, width int, height int)
}

type BoxFooter interface {
	tview.Primitive
}

type Boxed struct {
	BoxedContent
	options boxOptions
}

type boxOptions struct {
	leftBorder   bool
	leftPadding  int
	leftOffset   int
	rightBorder  bool
	rightPadding int
	rightOffset  int

	footer BoxFooter

	tabs []*PanelTab
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

func WithTabs(tabs ...*PanelTab) BoxOption {
	return func(opts *boxOptions) {
		opts.tabs = append(opts.tabs, tabs...)
	}
}

func WithFooter(footer BoxFooter) BoxOption {
	return func(opts *boxOptions) {
		opts.footer = footer
	}
}

type PanelTab struct {
	ID      string
	Title   string
	Hotkey  rune
	Checked bool
	Action  func(tab string)
}

func NewBoxed(inner BoxedContent, o ...BoxOption) *Boxed {
	b := Boxed{
		BoxedContent: inner,
	}
	for _, option := range o {
		option(&b.options)
	}
	inner.SetBorderPadding(1, 1, b.options.leftPadding, b.options.rightPadding)
	return &b
}

func (b Boxed) Draw(screen tcell.Screen) {
	b.BoxedContent.Draw(screen)
	b.drawBorders(screen)
}

func (b Boxed) drawBorders(screen tcell.Screen) {
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

	horizontalStart := x + b.options.leftOffset
	horizontalLen := width
	horizontalLen += b.options.rightOffset - b.options.leftOffset
	if hasFocus {
		if b.options.leftBorder {
			horizontalStart += 1
			horizontalLen -= 1
		}
		if b.options.rightBorder {
			horizontalLen -= 1
		}
	}

	type borderRendererFunc func(tcell.Screen, int, int, int)

	horizontalBorder := func(y int, contentWidth int, renderer any) {
		if contentWidth == 0 || renderer == nil {
			for i := 0; i < horizontalLen; i++ {
				screen.SetContent(horizontalStart+i, y, topLineChar, nil, lineStyle)
			}
			return
		}
		leftLen := (horizontalLen - contentWidth) / 2
		for i := 0; i < leftLen-1; i++ {
			screen.SetContent(horizontalStart+i, y, topLineChar, nil, lineStyle)
		}
		if hasFocus {
			screen.SetContent(horizontalStart+leftLen-1, y, '╡', nil, lineStyle)
		} else {
			screen.SetContent(horizontalStart+leftLen-1, y, '┤', nil, lineStyle)
		}

		contentStart := horizontalStart + leftLen
		switch content := renderer.(type) {
		case borderRendererFunc:
			content(screen, contentStart, y, contentWidth)
		case tview.Primitive:
			content.SetRect(contentStart, y, contentWidth, 1)
			content.Draw(screen)
		}

		rightStart := horizontalStart + leftLen + contentWidth
		if hasFocus {
			screen.SetContent(rightStart, y, '╞', nil, lineStyle)
		} else {
			screen.SetContent(rightStart, y, '├', nil, lineStyle)
		}
		rightLen := horizontalLen - leftLen - contentWidth
		for i := 1; i < rightLen; i++ {
			screen.SetContent(rightStart+i, y, topLineChar, nil, lineStyle)
		}
	}

	var title string
	if len(b.options.tabs) == 0 {
		title = b.GetTitle()
	} else {
		var sb strings.Builder
		for i, tab := range b.options.tabs {
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
					fmt.Sprintf("[#%06x]%c[-][DarkGray]", themes.CurrentTheme.HotkeyColor().Hex(), tab.Hotkey), 1)
			}
			title = fmt.Sprintf("[DarkGray]%s[-]", title)
			sb.WriteString(title)
		}
		title = sb.String()
	}

	titleWidth := tview.TaggedStringWidth(title)
	horizontalBorder(y, titleWidth, borderRendererFunc(func(screen tcell.Screen, x int, y int, width int) {
		tview.Print(screen, title, x, y, width, tview.AlignLeft, tcell.ColorGhostWhite)
	})) // top line

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

	if b.options.leftBorder {
		if hasFocus {
			verticalBorder(x+b.options.leftOffset, '╒', '╘')
		} else {
			verticalBorder(x+b.options.leftOffset, '┬', '┴')
		}
	}
	if b.options.rightBorder {
		if hasFocus {
			verticalBorder(x+width-1+b.options.rightOffset, '╕', '╛')
		} else {
			verticalBorder(x+width-1+b.options.rightOffset, '┬', '┴')
		}
	}

	if b.options.footer != nil {
		footerWidth := borderPrimitiveWidth(b.options.footer)
		horizontalBorder(y+height-1, footerWidth, b.options.footer) // bottom line with footer
	} else {
		horizontalBorder(y+height-1, 0, nil) // bottom line
	}
}

func borderPrimitiveWidth(footer tview.Primitive) int {
	if footer == nil {
		return 0
	}
	switch footerTyped := footer.(type) {
	case *tview.TextView:
		text := footerTyped.GetText(false)
		if newline := strings.IndexByte(text, '\n'); newline >= 0 {
			text = text[:newline]
		}
		return tview.TaggedStringWidth(text)
	default:
		_, _, width, _ := footer.GetRect()
		if width > 0 {
			return width
		}
		return 0
	}
}
