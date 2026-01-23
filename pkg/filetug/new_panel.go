package filetug

import (
	"context"
	"path"

	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type NewPanel struct {
	flex  *tview.Flex
	input *tview.InputField
	nav   *Navigator
	*sneatv.Boxed
}

func NewNewPanel(nav *Navigator) *NewPanel {
	p := &NewPanel{
		nav: nav,
	}

	p.input = tview.NewInputField().
		SetLabel("Name: ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetFieldTextColor(tview.Styles.PrimaryTextColor)

	createDirBtn := sneatv.NewButtonWithShortcut("Create directory", 'd').
		SetSelectedFunc(func() {
			p.createDir()
		})

	createFileBtn := sneatv.NewButtonWithShortcut("Create file", 'f').
		SetSelectedFunc(func() {
			p.createFile()
		})

	p.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.input, 1, 1, true).
		AddItem(nil, 1, 0, false).
		AddItem(createDirBtn, 1, 1, false).
		AddItem(nil, 1, 0, false).
		AddItem(createFileBtn, 1, 1, false)

	p.Boxed = sneatv.NewBoxed(p.flex,
		sneatv.WithLeftBorder(0, -1),
	)
	p.Boxed.SetTitle("New")

	p.input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			p.createFile()
		} else if key == tcell.KeyEsc {
			p.nav.right.SetContent(p.nav.dirSummary)
			p.nav.SetFocus()
		}
	})

	p.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if createDirBtn.HasFocus() {
				p.nav.setAppFocus(createFileBtn)
			} else if createFileBtn.HasFocus() {
				p.nav.setAppFocus(p.input)
			} else {
				p.nav.setAppFocus(createDirBtn)
			}
			return nil
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'd':
				if event.Modifiers()&tcell.ModAlt != 0 {
					return event
				}
				p.createDir()
				return nil
			case 'f':
				if event.Modifiers()&tcell.ModAlt != 0 {
					return event
				}
				p.createFile()
				return nil
			}
		}
		return event
	})

	return p
}

func (p *NewPanel) Show() {
	p.input.SetText("")
	p.nav.right.SetContent(p)
	p.nav.setAppFocus(p)
}

func (p *NewPanel) Focus(delegate func(p tview.Primitive)) {
	p.nav.activeCol = 2
	delegate(p.input)
}

func (p *NewPanel) createDir() {
	name := p.input.GetText()
	if name == "" {
		return
	}
	currentDir := p.nav.current.dir
	fullPath := path.Join(currentDir, name)

	err := p.nav.store.CreateDir(context.Background(), fullPath)
	if err != nil {
		// TODO: show error
		return
	}

	p.nav.right.SetContent(p.nav.dirSummary)
	p.nav.goDir(fullPath)
}

func (p *NewPanel) createFile() {
	name := p.input.GetText()
	if name == "" {
		return
	}
	currentDir := p.nav.current.dir
	fullPath := path.Join(currentDir, name)

	err := p.nav.store.CreateFile(context.Background(), fullPath)
	if err != nil {
		// TODO: show error
		return
	}

	p.nav.right.SetContent(p.nav.dirSummary)
	p.nav.showDir(context.Background(), p.nav.dirsTree.rootNode, currentDir, false)
	p.nav.files.SetCurrentFile(name)
	p.nav.setAppFocus(p.nav.files.Boxed)
}
