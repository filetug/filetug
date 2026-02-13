package filetug

import (
	"path"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
)

func (t *Tree) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRight:
		t.nav.app.SetFocus(t.nav.files)
		return nil
	case tcell.KeyLeft:
		currentNode := t.tv.GetCurrentNode()
		refValue := currentNode.GetReference()
		switch ref := refValue.(type) {
		case *files.DirContext:
			parentDir, _ := path.Split(ref.Path())
			parentContext := files.NewDirContext(t.nav.store, parentDir, nil)
			t.nav.goDir(parentContext)
			return nil
		}
		return event
	case tcell.KeyEnter:
		currentNode := t.tv.GetCurrentNode()
		switch ref := currentNode.GetReference().(type) {
		case *files.DirContext:
			dirPath := ref.Path()
			if dirPath != "/" {
				dirPath = strings.TrimSuffix(dirPath, "/")
			}
			if currentNode == t.tv.GetRoot() {
				expandedRef := fsutils.ExpandHome(dirPath)
				var parentDir string
				parentDir, _ = path.Split(expandedRef)
				dirPath = parentDir
			}
			dirContext := files.NewDirContext(t.nav.store, dirPath, nil)
			t.nav.goDir(dirContext)
			return nil
		}
		return event
	case tcell.KeyUp:
		if t.tv.GetCurrentNode() == t.tv.GetRoot() {
			t.nav.breadcrumbs.TakeFocus(t.tv)
			t.nav.app.SetFocus(t.nav.breadcrumbs)
			return nil
		}
		return event
	case tcell.KeyBackspace:
		t.SetSearch(t.searchPattern[:len(t.searchPattern)-1])
		return nil
	case tcell.KeyEscape:
		t.SetSearch("")
		return nil
	case tcell.KeyRune:
		if event = t.nav.globalNavInputCapture(event); event == nil {
			return nil
		}
		s := string(event.Rune())
		if t.searchPattern == "" && s == " " {
			return event
		}
		lower := strings.ToLower(s)
		t.SetSearch(t.searchPattern + lower)
		return nil
	default:
		return event
	}
}
