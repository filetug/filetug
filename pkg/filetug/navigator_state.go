package filetug

import (
	"net/url"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/ftpfile"
	"github.com/filetug/filetug/pkg/files/httpfile"
)

func initNavigatorWithPersistedState(nav *Navigator) {
	if state, stateErr := getState(); state != nil {
		if state.Store == "" {
			state.Store = "file:"
		}
		schema := state.Store
		if i := strings.Index(state.Store, ":"); i >= 0 {
			schema = state.Store[:i]
		}
		switch schema {
		case "http", "https":
			root, err := url.Parse(state.Store)
			if err == nil {
				nav.store = httpfile.NewStore(*root)
			}
		case "ftp":
			root, err := url.Parse(state.Store)
			if err == nil {
				store := ftpfile.NewStore(*root)
				if store != nil {
					nav.store = store
				}
			}
		}

		if state.CurrentDir == "" {
			state.CurrentDir = "~"
		}
		dirPath := state.CurrentDir
		if strings.HasPrefix(state.CurrentDir, "https://") {
			currentUrl, err := url.Parse(state.CurrentDir)
			if err != nil {
				return
			}
			dirPath = currentUrl.Path
			currentUrl.Path = "/"
			nav.store = httpfile.NewStore(*currentUrl)
		}
		dirContext := files.NewDirContext(nav.store, dirPath, nil)
		nav.goDir(dirContext)
		if stateErr == nil {
			if state.CurrentDirEntry != "" {
				nav.files.SetCurrentFile(state.CurrentDirEntry)
			}
		}
	}
}
