package filetug

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func showHelpModal(nav *Navigator) {
	modal, _, _ := createHelpModal(nav, nav.Flex)
	nav.setAppRoot(modal, true)
}

func createHelpModal(nav *Navigator, root tview.Primitive) (modal tview.Primitive, helpView *tview.TextView, button *tview.Button) {
	const helpText = `F1 - Help
Alt+F - Favorites
Alt+G - Go to...
Al+P - Show/Hide previewer
Alt+C - Copy filesPanel & directories
Alt+M - Move filesPanel & directories
Alt+D - Delete filesPanel & directories
Alt+V - View file
Alt+E - Edit file
Alt+X - Exit the app`

	helpView = tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetTextAlign(tview.AlignCenter)

	helpView.SetBackgroundColor(tcell.ColorDarkBlue)
	helpView.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter)

	// Create a modal-like layout using a Grid to center the helpView
	// Close function
	closeHelp := func() {
		nav.setAppRoot(root, true)
		nav.setAppFocus(nav.dirsTree.TreeView)
	}

	helpView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF1 {
			closeHelp()
			return nil
		}
		return event
	})

	// Add a button to close
	button = tview.NewButton("Close").SetSelectedFunc(closeHelp)
	button.SetBackgroundColor(tcell.ColorDarkBlue)
	button.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF1 {
			closeHelp()
			return nil
		}
		return event
	})

	// Update helpView to include the button or use a Flex
	helpFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(helpView, 0, 1, false).
		AddItem(button, 1, 0, true)

	helpFlex.SetBorder(true).
		SetTitle(" FileTug - Help ").
		SetTitleAlign(tview.AlignCenter)
	helpFlex.SetBackgroundColor(tcell.ColorDarkBlue)
	helpView.SetBorder(false) // Border is now on helpFlex

	// Update modal to use helpFlex
	modal = tview.NewGrid().
		SetColumns(0, 40, 0).
		SetRows(0, 13, 0).
		AddItem(helpFlex, 1, 1, 1, 1, 0, 0, true)

	return modal, helpView, button
}
