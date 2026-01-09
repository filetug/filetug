package filetug

import (
	"fmt"

	"github.com/rivo/tview"
)

func Main() {
	app := tview.NewApplication()
	app.EnableMouse(true)
	app.SetRoot(NewNavigator(app), true)
	err := app.Run()
	if err != nil {
		fmt.Print(err)
	}
}
