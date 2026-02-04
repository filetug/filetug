package filetug

import (
	"context"
	"path"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type NewPanel struct {
	flex          *tview.Flex
	input         *tview.InputField
	createDirBtn  *sneatv.ButtonWithShortcut
	createFileBtn *sneatv.ButtonWithShortcut
	nav           *Navigator
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

	createDirBtn := sneatv.NewButtonWithShortcut("Create directory", 'd')
	createDirBtn.SetSelectedFunc(func() {
		p.createDir()
	})

	createFileBtn := sneatv.NewButtonWithShortcut("Create file", 'f')
	createFileBtn.SetSelectedFunc(func() {
		p.createFile()
	})

	p.createDirBtn = createDirBtn
	p.createFileBtn = createFileBtn

	p.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.input, 1, 1, true).
		AddItem(nil, 1, 0, false).
		AddItem(createDirBtn, 1, 1, false).
		AddItem(nil, 1, 0, false).
		AddItem(createFileBtn, 1, 1, false)

	p.Boxed = sneatv.NewBoxed(p.flex,
		sneatv.WithLeftBorder(0, -1),
	)
	p.SetTitle("New")

	p.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			p.createFile()
		case tcell.KeyEscape:
			p.nav.right.SetContent(p.nav.previewer)
			p.nav.SetFocus()
		}
	})

	p.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if p.createDirBtn.HasFocus() {
				p.nav.app.SetFocus(p.createFileBtn)
			} else if p.createFileBtn.HasFocus() {
				p.nav.app.SetFocus(p.input)
			} else {
				p.nav.app.SetFocus(p.createDirBtn)
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
	p.nav.app.SetFocus(p)
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
	currentDir := p.nav.currentDirPath()
	if currentDir == "" {
		return
	}
	fullPath := path.Join(currentDir, name)

	ctx := context.Background()
	err := p.nav.store.CreateDir(ctx, fullPath)
	if err != nil {
		// TODO: show error
		return
	}

	p.nav.right.SetContent(p.nav.previewer)
	dirContext := files.NewDirContext(p.nav.store, fullPath, nil)
	p.nav.app.QueueUpdateDraw(func() {
		p.nav.goDir(dirContext)
	})
}

func (p *NewPanel) createFile() {
	name := p.input.GetText()
	if name == "" {
		return
	}
	currentDir := p.nav.currentDirPath()
	if currentDir == "" {
		return
	}
	fullPath := path.Join(currentDir, name)

	ctx := context.Background()
	err := p.nav.store.CreateFile(ctx, fullPath)
	if err != nil {
		// TODO: show error
		return
	}

	p.nav.right.SetContent(p.nav.previewer)
	ctx = context.Background()
	dirContext := files.NewDirContext(p.nav.store, currentDir, nil)
	p.nav.showDir(ctx, p.nav.dirsTree.rootNode, dirContext, false)
	p.nav.files.SetCurrentFile(name)
	p.nav.app.SetFocus(p.nav.files.Boxed)
}
