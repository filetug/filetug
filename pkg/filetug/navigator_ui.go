package filetug

import (
	"path"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (nav *Navigator) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	if nav.app == nil {
		return event
	}
	nav.bottom.isCtrl = event.Modifiers()&tcell.ModCtrl != 0
	switch event.Key() {
	case tcell.KeyF1:
		showHelpModal(nav)
		return nil
	case tcell.KeyF7:
		nav.showNewPanel()
		return nil
	case tcell.KeyF8:
		nav.delete()
		return nil
	case tcell.KeyF10:
		nav.showScriptsPanel()
		return nil
	case tcell.KeyRune:
		if event.Modifiers()&tcell.ModAlt != 0 {
			switch r := event.Rune(); r {
			case 'f', 'F':
				nav.ShowFavorites()
				return nil
			case 'm', 'M':
				nav.showMasks()
				return nil
			case '0':
				copy(nav.proportions, defaultProportions)
				nav.createColumns()
			case '+', '=':
				nav.resize(increase)
				return nil
			case '-', '_':
				nav.resize(decrease)
				return nil
			case '/', 'r', 'R':
				nav.goRoot()
				return nil
			case '~', '`', 'h', 'H':
				nav.goHome()
				return nil
			case 'x', 'X':
				nav.app.Stop()
				return nil
			default:
				return event
			}
		}
		return event
	default:
		return event
	}
}

// globalNavInputCapture should be invoked only from specific boxes like Tree and filesPanel.
func (nav *Navigator) globalNavInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if nav.app == nil {
		return event
	}
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case '/':
			nav.goRoot()
			return nil
		case '`':
			nav.goHome()
			return nil
		}
	}
	return event
}

func (nav *Navigator) resize(mode resizeMode) {
	switch nav.activeCol {
	case 0:
		nav.proportions[0] += 2 * int(mode)
		nav.proportions[1] -= 1 * int(mode)
		nav.proportions[2] -= 1 * int(mode)
	case 1:
		nav.proportions[0] -= 1 * int(mode)
		nav.proportions[1] += 2 * int(mode)
		nav.proportions[2] -= 1 * int(mode)
	case 2:
		nav.proportions[0] -= 1 * int(mode)
		nav.proportions[1] -= 1 * int(mode)
		nav.proportions[2] += 2 * int(mode)
	default:
		return
	}

	nav.createColumns()
}

func (nav *Navigator) setBreadcrumbs() {
	if nav.breadcrumbs == nil {
		return
	}
	nav.breadcrumbs.Clear()

	if nav.store == nil {
		return
	}
	rootPath := nav.store.RootURL().Path
	if rootPath == "" {
		rootPath = "/"
	}
	{
		rootTitle := nav.store.RootTitle()
		rootTitle = strings.TrimSuffix(rootTitle, "/")
		rootBreadcrumb := crumbs.NewBreadcrumb(rootTitle, func() error {
			dirContext := files.NewDirContext(nav.store, rootPath, nil)
			nav.goDir(dirContext)
			return nil
		})
		nav.breadcrumbs.Push(rootBreadcrumb)
	}

	currentDirPath := nav.currentDirPath()
	if currentDirPath == "" {
		return
	}
	trimmedDir := strings.TrimPrefix(currentDirPath, rootPath)
	relativePath := strings.TrimPrefix(trimmedDir, "/")
	if relativePath == "" {
		return
	}
	relativePath = strings.TrimSuffix(relativePath, "/")
	currentDir := strings.Split(relativePath, "/")
	breadPaths := make([]string, 0, len(currentDir))
	breadPaths = append(breadPaths, rootPath)
	for _, p := range currentDir {
		if p == "" {
			p = "{EMPTY PATH ITEM}"
		}
		breadPaths = append(breadPaths, p)
		breadPath := path.Join(breadPaths...)
		breadcrumb := crumbs.NewBreadcrumb(p, func() error {
			dirContext := files.NewDirContext(nav.store, breadPath, nil)
			nav.goDir(dirContext)
			return nil
		})
		nav.breadcrumbs.Push(breadcrumb)
	}
}

func (nav *Navigator) showNodeError(node *tview.TreeNode, err error) {
	if node == nil {
		return
	}
	if nav.dirsTree != nil {
		nav.dirsTree.setError(node, err)
	}
	if nav.files != nil {
		dirRecords := NewFileRows(files.NewDirContext(nav.store, getNodePath(node), nil))
		nav.files.SetRows(dirRecords, false)
	}
	if nav.previewer != nil {
		text := err.Error()
		nav.previewer.textView.SetText(text).SetWrap(true).SetTextColor(tcell.ColorOrangeRed)
	}
	if nav.right != nil {
		nav.right.SetContent(nav.previewer)
	}
}
