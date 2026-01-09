package filetug

import (
	"fmt"

	"github.com/rivo/tview"
)

func Main() {
	app := tview.NewApplication()
	SetupApp(app)
	err := app.Run()
	if err != nil {
		fmt.Print(err)
	}
}

func SetupApp(app *tview.Application) {
	app.EnableMouse(true)
	app.SetRoot(NewNavigator(app), true)
}
