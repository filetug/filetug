package crumbs

import (
	"net/url"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Breadcrumbs struct {
	*tview.Box
	items             []Breadcrumb
	separator         string
	separatorStartIdx int
	selectedItemIndex int             // -1 means last item is considered focused
	nextFocusTarget   tview.Primitive // optional: where to move focus on Tab/Down
	prevFocusTarget   tview.Primitive // optional: where to move focus on Shift+Tab/Up
}

func (b *Breadcrumbs) GoHome() error {
	return b.items[0].Action()
}

func (b *Breadcrumbs) TakeFocus(from tview.Primitive) {
	b.nextFocusTarget = from
	b.selectedItemIndex -= 1
}

func (b *Breadcrumbs) SelectedItemIndex() int {
	return b.selectedItemIndex
}

func (b *Breadcrumbs) ItemsCount() int {
	return len(b.items)
}

func (b *Breadcrumbs) IsLastItemSelected() bool {
	return b.selectedItemIndex == b.ItemsCount()-1
}

func NewBreadcrumbs(root Breadcrumb, options ...func(bc *Breadcrumbs)) *Breadcrumbs {
	b := &Breadcrumbs{
		Box:               tview.NewBox(),
		items:             make([]Breadcrumb, 0, 8),
		separator:         " > ",
		selectedItemIndex: -1,
	}
	b.items = append(b.items, root)
	for _, o := range options {
		o(b)
	}
	return b
}

func (b *Breadcrumbs) Push(item Breadcrumb) {
	b.selectedItemIndex = len(b.items)
	b.items = append(b.items, item)
}

func (b *Breadcrumbs) Clear() {
	b.items = b.items[:1]
}

func (b *Breadcrumbs) Draw(screen tcell.Screen) {
	// Draw the base box (background, border, title, etc.).
	b.DrawForSubclass(screen, b)

	// Get the inner drawing area and keep text on a single header line.
	x, y, width, _ := b.GetInnerRect()
	if width <= 0 {
		return
	}

	// Determine which item is focused. Default to last.
	focus := b.selectedItemIndex
	if focus < 0 || focus >= len(b.items) {
		focus = len(b.items) - 1
	}

	// Draw items horizontally within the header row.
	cursorX := x
	maxX := x + width
	for i, item := range b.items {
		if cursorX >= maxX {
			break
		}
		title := strings.TrimSuffix(item.GetTitle(), b.separator)
		var isURL bool
		if u, err := url.Parse(title); err == nil && u.Scheme != "" {
			isURL = true
		}
		label := tview.Escape(title)
		var text string
		if i == focus {
			if b.HasFocus() {
				// Highlight selected item with background color when focused.
				text = "[black:yellow]" + label + "[-:-:-]"
			} else {
				// Focused item (last by default) remains normal when not focused.
				text = label
			}
		} else {
			// Dim for unfocused items.
			text = "[::d]" + label + "[-:-:-]"
		}
		color := item.GetColor()
		if color == 0 {
			color = tcell.ColorYellow
		}
		_, printed := tview.Print(screen, text, cursorX, y, maxX-cursorX, tview.AlignLeft, color)
		cursorX += printed
		if isURL {
			tview.Print(screen, " ", cursorX, y, 1, tview.AlignLeft, color)
			cursorX++
		}
		// Add a separator between items if there is still room.
		if i >= b.separatorStartIdx && i < len(b.items)-1 && cursorX < maxX {
			_, sp := tview.Print(screen, b.separator, cursorX, y, maxX-cursorX, tview.AlignLeft, tcell.ColorGray)
			cursorX += sp
		}
	}
}

// SetNextFocusTarget sets the primitive to focus when Tab/Down is pressed.
func (b *Breadcrumbs) SetNextFocusTarget(p tview.Primitive) *Breadcrumbs {
	b.nextFocusTarget = p
	return b
}

// SetPrevFocusTarget sets the primitive to focus when Shift+Tab/Up is pressed.
func (b *Breadcrumbs) SetPrevFocusTarget(p tview.Primitive) *Breadcrumbs {
	b.prevFocusTarget = p
	return b
}

// Focus is called when this primitive receives focus.
func (b *Breadcrumbs) Focus(delegate func(p tview.Primitive)) {
	// When receiving focus, keep current selection if valid; otherwise select the last item.
	if len(b.items) > 0 && (b.selectedItemIndex < 0 || b.selectedItemIndex >= len(b.items)-1) {
		b.selectedItemIndex = len(b.items) - 2
	}
	b.Box.Focus(delegate)
}

// Blur is called when this primitive loses focus.
func (b *Breadcrumbs) Blur() {
	b.selectedItemIndex = len(b.items) - 1
	b.Box.Blur()
}

func (b *Breadcrumbs) defaultInputHandler(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	//if b.inputHandler != nil {
	//	if event = b.inputHandler(b, event, setFocus); event == nil {
	//		return
	//	}
	//}
	if len(b.items) == 0 {
		return
	}
	sel := b.selectedItemIndex
	if sel < 0 || sel >= len(b.items) {
		sel = len(b.items) - 1
	}
	switch event.Key() {
	case tcell.KeyLeft:
		if sel > 0 {
			b.selectedItemIndex = sel - 1
		}
		return
	case tcell.KeyRight:
		if sel < len(b.items)-1 {
			b.selectedItemIndex = sel + 1
		}
		// If already at last item, do nothing (consume key, keep focus).
		return
	case tcell.KeyEnter:
		if sel >= 0 && sel < len(b.items) && b.items[sel] != nil {
			_ = b.items[sel].Action()
		}
		return
	case tcell.KeyRune:
		switch event.Rune() {
		case '<':
			if sel > 0 {
				b.selectedItemIndex = sel - 1
			}
			return
		case '>':
			if sel < len(b.items)-1 {
				b.selectedItemIndex = sel + 1
			}
			return
		}
	case tcell.KeyTab, tcell.KeyDown:
		if b.nextFocusTarget != nil {
			// On blur we always highlight the last item
			b.selectedItemIndex = len(b.items) - 1
			setFocus(b.nextFocusTarget)
			setFocus(b.nextFocusTarget)
		} else {
			setFocus(nil)
		}
		return
	//case tcell.KeyBacktab, tcell.KeyUp:
	//	if b.prevFocusTarget != nil {
	//		setFocus(b.prevFocusTarget)
	//	} else {
	//		setFocus(nil)
	//	}
	//	return
	default:
		return
	}
}

// InputHandler handles keyboard input for navigation and activation.
func (b *Breadcrumbs) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.WrapInputHandler(b.defaultInputHandler)
}

// MouseHandler handles selection via mouse and focusing on click.
func (b *Breadcrumbs) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return b.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if action != tview.MouseLeftDown && action != tview.MouseLeftClick {
			return false, nil
		}
		x, y := event.Position()
		if !b.InInnerRect(x, y) {
			return false, nil
		}
		// Determine which item was clicked based on x coordinate.
		innerX, innerY, width, _ := b.GetInnerRect()
		_ = innerY
		cursorX := innerX
		maxX := innerX + width
		for i, item := range b.items {
			if cursorX >= maxX {
				break
			}
			label := tview.Escape(item.GetTitle())
			// Compute width based on rune count (no style tags in label after Escape).
			w := len([]rune(label))
			if x >= cursorX && x < cursorX+w {
				b.selectedItemIndex = i
				// Focus breadcrumbs so it highlights selection.
				setFocus(b)
				// Call action only on mouse click (not on mouse down) to avoid double invocation.
				if action == tview.MouseLeftClick && b.items[i] != nil {
					_ = b.items[i].Action()
				}
				return true, nil
			}
			cursorX += w
			// Separator width
			if i < len(b.items)-1 && cursorX < maxX {
				sep := b.separator
				sepW := len([]rune(sep))
				cursorX += sepW
			}
		}
		setFocus(b)
		return true, nil
	})
}
