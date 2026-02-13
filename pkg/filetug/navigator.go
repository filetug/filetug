package filetug

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/filetug/masks"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/rivo/tview"
	"github.com/strongo/strongo-tui/pkg/colors"
)

type Navigator struct {
	app navigator.App

	// Deprecated, use Navigator.app.QueueUpdateDraw
	//queueUpdateDraw func(f func()) // replaced with mock in tests

	// Deprecated, use Navigator.app.SetFocus
	//setAppFocus func(p tview.Primitive) // replaced with mock in tests

	// Deprecated, use Navigator.app.SetRoot
	//setAppRoot func(root tview.Primitive, fullscreen bool) // replaced with mock in tests

	// Deprecated
	//stopApp func()

	// o - navigatorOptions
	o navigatorOptions

	store files.Store

	breadcrumbs *crumbs.Breadcrumbs

	*tview.Flex
	main *tview.Flex

	current ftstate.Current
	prev    ftstate.Current

	activeCol   int
	proportions []int

	filesSelectionChangedFunc func(row, column int)

	favoritesFocusFunc func()

	previewerFocusFunc func()
	previewerBlurFunc  func()

	left  *Container
	right *Container

	dirsTree  *Tree
	favorites *favoritesPanel
	masks     *masks.Panel
	newPanel  *NewPanel

	files *filesPanel

	// dirSummary *viewers.DirPreviewer - we do not want this anymore as it's part of the previewerPanel now.

	previewer *previewerPanel

	bottom *bottom

	saveCurrentDir func(store, currentDir string)

	gitStatusCache   map[string]*gitutils.RepoStatus
	gitStatusCacheMu sync.RWMutex
	cancel           context.CancelFunc

	showError func(err error)
}

func (nav *Navigator) SetStore(store files.Store) {
	nav.store = store
	nav.dirsTree.onStoreChange()
	nav.files.onStoreChange()
}

func (nav *Navigator) SetFocus() {
	if nav.app != nil {
		nav.app.SetFocus(nav.dirsTree.tv)
	}
}

func (nav *Navigator) SetFocusToContainer(index int) {
	if nav.app == nil {
		return
	}
	switch index {
	case nav.left.index:
		nav.app.SetFocus(nav.left.Flex)
	case nav.right.index:
		nav.app.SetFocus(nav.right.Flex)
	case 1:
		nav.app.SetFocus(nav.files.Boxed)
	}
}

type navigatorOptions struct {
	moveFocusUp            func(source tview.Primitive)
	skipAsyncFavoritesLoad bool
}

type NavigatorOption func(o *navigatorOptions)

func OnMoveFocusUp(f func(source tview.Primitive)) NavigatorOption {
	return func(o *navigatorOptions) {
		o.moveFocusUp = f
	}
}

var getState = ftstate.GetState
var getDirStatus = gitutils.GetDirStatus
var getFileStatus = gitutils.GetFileStatus

func NewNavigator(app navigator.App, options ...NavigatorOption) *Navigator {
	if app == nil {
		panic("app cannot be nil")
	}
	defaultStore := osfile.NewStore("/")
	rootBreadcrumb := crumbs.NewBreadcrumb("FileTug: ", func() error {
		return nil
	})
	rootBreadcrumb.SetColor(colors.TableHeaderColor)
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	nav := &Navigator{
		app:   app,
		store: defaultStore,
		breadcrumbs: crumbs.NewBreadcrumbs(
			rootBreadcrumb,
			crumbs.WithSeparator("/"),
			crumbs.WithSeparatorStartIndex(1),
		),
		Flex:           flex,
		main:           tview.NewFlex(),
		proportions:    make([]int, 3),
		gitStatusCache: make(map[string]*gitutils.RepoStatus),
		saveCurrentDir: ftstate.SaveCurrentDir,
		showError: func(err error) {
			log.Println(err) // TODO(help-wanted): Show error to user
		},
	}
	nav.bottom = newBottom(nav)
	nav.right = NewContainer(2, nav)
	nav.favorites = newFavoritesPanel(nav)
	nav.dirsTree = NewTree(nav)
	nav.newPanel = NewNewPanel(nav)
	nav.AddItem(nav.breadcrumbs, 1, 0, false)

	copy(nav.proportions, defaultProportions)

	nav.files = newFiles(nav)
	nav.previewer = newPreviewerPanel(nav)

	nav.right.SetContent(nav.previewer)

	for _, option := range options {
		option(&nav.o)
	}

	createLeft(nav)

	nav.AddItem(nav.main, 0, 1, true)

	nav.AddItem(nav.bottom, 1, 0, false)

	nav.createColumns()

	if !nav.o.skipAsyncFavoritesLoad {
		nav.favorites.loadUserFavorites()
	}

	return nav
}

var defaultProportions = []int{5, 12, 7}

func (nav *Navigator) NewDirContext(path string, children []os.DirEntry) *files.DirContext {
	return files.NewDirContext(nav.store, path, children)
}

func (nav *Navigator) createColumns() {

	nav.main.Clear()
	nav.main.AddItem(nav.left, 0, nav.proportions[0], true)
	nav.main.AddItem(nil, 1, 0, false)
	nav.main.AddItem(nav.files, 0, nav.proportions[1], true)
	nav.main.AddItem(nil, 1, 0, false)
	nav.main.AddItem(nav.right, 0, nav.proportions[2], true)

	nav.SetInputCapture(nav.inputCapture)
}

type resizeMode int

const (
	increase resizeMode = 1
	decrease resizeMode = -1
)

func (nav *Navigator) getCurrentBrowser() browser {
	switch nav.activeCol {
	case 0:
		return nav.dirsTree
	case 1:
		return nav.files
	}
	return nil
}

// TODO: Get rid of package level var saveCurrentDir - see NewNavigator for proper implementation
var saveCurrentDir = ftstate.SaveCurrentDir

func (nav *Navigator) currentDirPath() string {
	currentDir := nav.current.Dir()
	if currentDir == nil {
		return ""
	}
	return currentDir.Path()
}
