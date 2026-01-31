package filetug

import (
	"github.com/filetug/filetug/pkg/filetug/navigator"
)

func SetupApp(app navigator.App) {
	app.EnableMouse(true)
	nav := NewNavigator(app)
	initNavigatorWithPersistedState(nav)
	app.SetRoot(nav, true)
}
