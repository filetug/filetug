package filetug

import (
	"github.com/rivo/tview"
)

func SetupApp(app *tview.Application) {
	app.EnableMouse(true)
	nav := NewNavigator(app)
	app.SetRoot(nav, true)
}
