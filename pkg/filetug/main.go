package filetug

import (
	"path/filepath"

	"github.com/filetug/filetug/pkg/filetug/navigator"
)

func SetupApp(app navigator.App) {
	SetupAppAtPath(app, "")
}

func SetupAppAtPath(app navigator.App, initialPath string) {
	app.EnableMouse(true)
	nav := NewNavigator(app)
	if initialPath == "" {
		initNavigatorWithPersistedState(nav)
	} else if absolutePath, err := filepath.Abs(initialPath); err == nil {
		nav.goDirByPath(absolutePath)
	}
	app.SetRoot(nav, true)
}
