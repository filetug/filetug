package filetug

import (
	"context"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/datatug/filetug/pkg/files"
	"github.com/datatug/filetug/pkg/files/ftpfile"
	"github.com/datatug/filetug/pkg/files/httpfile"
	"github.com/datatug/filetug/pkg/files/osfile"
	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/datatug/filetug/pkg/ftstate"
	"github.com/datatug/filetug/pkg/gitutils"
	"github.com/datatug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Navigator struct {
	app *tview.Application
	o   navigatorOptions

	store files.Store

	breadcrumbs *sneatv.Breadcrumbs

	*tview.Flex
	main *tview.Flex

	current     current
	activeCol   int
	proportions []int

	filesSelectionChangedFunc func(row, column int)

	favoritesFocusFunc func()

	previewerFocusFunc func()
	previewerBlurFunc  func()

	left  *container
	right *container

	dirsTree  *Tree
	favorites *favorites

	files *filesPanel

	dirSummary *dirSummary

	previewer *previewer

	bottom *bottom

	gitStatusCache   map[string]*gitutils.RepoStatus
	gitStatusCacheMu sync.RWMutex
	cancel           context.CancelFunc
}

func (nav *Navigator) SetStore(store files.Store) {
	nav.store = store
	nav.dirsTree.onStoreChange()
	nav.files.onStoreChange()
}

func (nav *Navigator) SetFocus() {
	nav.app.SetFocus(nav.dirsTree.TreeView)
}

func (nav *Navigator) SetFocusToContainer(index int) {
	switch index {
	case nav.left.index:
		nav.app.SetFocus(nav.left.Flex)
	case nav.right.index:
		nav.app.SetFocus(nav.right.Flex)
	case 1:
		nav.app.SetFocus(nav.files.boxed)
	}
}

type navigatorOptions struct {
	moveFocusUp func(source tview.Primitive)
}

type NavigatorOption func(o *navigatorOptions)

func OnMoveFocusUp(f func(source tview.Primitive)) NavigatorOption {
	return func(o *navigatorOptions) {
		o.moveFocusUp = f
	}
}

func NewNavigator(app *tview.Application, options ...NavigatorOption) *Navigator {

	nav := &Navigator{
		app:   app,
		store: osfile.NewStore("/"),
		breadcrumbs: sneatv.NewBreadcrumbs(
			sneatv.NewBreadcrumb("FileTug: ", func() error {
				return nil
			}).SetColor(tcell.ColorWhiteSmoke),
			sneatv.WithSeparator("/"),
			sneatv.WithSeparatorStartIndex(1),
		),
		Flex:           tview.NewFlex().SetDirection(tview.FlexRow),
		main:           tview.NewFlex(),
		bottom:         newBottom(),
		proportions:    make([]int, 3),
		gitStatusCache: make(map[string]*gitutils.RepoStatus),
	}
	nav.right = newContainer(2, nav)
	nav.favorites = newFavorites(nav)
	nav.dirsTree = NewTree(nav)
	nav.AddItem(nav.breadcrumbs, 1, 0, false)

	copy(nav.proportions, defaultProportions)

	nav.files = newFiles(nav)
	nav.previewer = newPreviewer(nav)
	nav.dirSummary = newDirSummary(nav)
	nav.right.SetContent(nav.dirSummary)

	for _, option := range options {
		option(&nav.o)
	}

	createLeft(nav)

	nav.AddItem(nav.main, 0, 1, true)

	nav.AddItem(nav.bottom, 1, 0, false)

	nav.createColumns()

	if state, stateErr := ftstate.GetState(); state != nil {
		if state.Store == "" {
			state.Store = "file:"
		}
		var schema string
		if i := strings.Index(state.Store, ":"); i < 0 {
			schema = state.Store
		}
		switch schema {
		case "http", "https":
			root, _ := url.Parse(state.Store)
			nav.store = httpfile.NewStore(*root)
		case "ftp":
			root, _ := url.Parse(state.Store)
			nav.store = ftpfile.NewStore(*root)
		}

		if state.CurrentDir == "" {
			state.CurrentDir = "~"
		}
		dirPath := state.CurrentDir
		if strings.HasPrefix(state.CurrentDir, "https://") {
			currentUrl, err := url.Parse(state.CurrentDir)
			if err != nil {
				return nil
			}
			dirPath = currentUrl.Path
			currentUrl.Path = "/"
			nav.store = httpfile.NewStore(*currentUrl)
		}
		nav.goDir(dirPath)
		if stateErr == nil {
			if state.CurrentDirEntry != "" {
				nav.files.SetCurrentFile(state.CurrentDirEntry)
			}
		}
	}

	return nav
}

var defaultProportions = []int{6, 10, 8}

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

func (nav *Navigator) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF1:
		showHelpModal(nav)
		return nil
	case tcell.KeyRune:
		if event.Modifiers()&tcell.ModAlt != 0 {
			switch r := event.Rune(); r {
			case 'f':
				nav.favorites.ShowFavorites()
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
				nav.goDir("/")
				return nil
			case '~', 'h', 'H':
				nav.goDir("~")
				return nil
			case 'x':
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

func (nav *Navigator) goDir(dir string) {
	ctx := context.Background()
	nav.dirsTree.setCurrentDir(dir)
	nav.showDir(ctx, nav.dirsTree.rootNode, dir, true)
	root := nav.store.RootURL()
	saveCurrentDir(root.String(), dir)
}

func (nav *Navigator) updateGitStatus(ctx context.Context, fullPath string, node *tview.TreeNode, prefix string) {
	nav.gitStatusCacheMu.RLock()
	cachedStatus, ok := nav.gitStatusCache[fullPath]
	nav.gitStatusCacheMu.RUnlock()

	if ok && node != nil {
		nav.app.QueueUpdateDraw(func() {
			node.SetText(prefix + cachedStatus.String())
		})
		return
	}

	status := gitutils.GetRepositoryStatus(ctx, fullPath)
	if status == nil {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	nav.gitStatusCacheMu.Lock()
	nav.gitStatusCache[fullPath] = status
	nav.gitStatusCacheMu.Unlock()

	nav.app.QueueUpdateDraw(func() {
		node.SetText(prefix + status.String())
	})
}

var saveCurrentDir = ftstate.SaveCurrentDir

// showDir updates all panels.
// The `isTreeRootChanged bool` argument is needed do distinguish root dir change from
// the case we simply select the root node in the tree.
func (nav *Navigator) showDir(ctx context.Context, node *tview.TreeNode, dir string, isTreeRootChanged bool) {

	nav.current.dir = fsutils.ExpandHome(dir)
	if node != nil {
		node.SetReference(nav.current.dir)
	}
	if nav.store.RootURL().Scheme == "file" {
		name, _ := path.Split(dir)
		go nav.updateGitStatus(ctx, nav.current.dir, node, name)
	}

	nav.setBreadcrumbs()
	nav.right.SetContent(nav.dirSummary)
	nav.previewer.textView.SetText("").SetTextColor(tcell.ColorWhiteSmoke)

	go func() {
		dirContext, err := nav.getDirData(ctx)
		nav.app.QueueUpdateDraw(func() {
			if err != nil {
				nav.showNodeError(node, err)
				return
			}
			nav.onDataLoaded(node, dirContext, isTreeRootChanged)
		})
	}()
}

func (nav *Navigator) onDataLoaded(node *tview.TreeNode, dirContext *DirContext, isTreeRootChanged bool) {
	nav.dirSummary.SetDir(dirContext)

	//nav.filesPanel.Clear()
	nav.files.table.SetSelectable(true, false)

	dirRecords := NewFileRows(dirContext)
	nav.files.SetRows(dirRecords, node != nav.dirsTree.rootNode)

	ctx := context.Background() // TODO: use a cancelable context
	if isTreeRootChanged {
		nav.dirsTree.setDirContext(ctx, node, dirContext)
	}
}

func (nav *Navigator) showNodeError(node *tview.TreeNode, err error) {
	if node == nil {
		return
	}
	nav.dirsTree.setError(node, err)
	dirRecords := NewFileRows(&DirContext{
		Store: nav.store,
		Path:  getNodePath(node),
	})
	nav.files.SetRows(dirRecords, false)
	text := err.Error()
	nav.previewer.textView.SetText(text).SetWrap(true).SetTextColor(tcell.ColorOrangeRed)
	nav.right.SetContent(nav.previewer)
}

func (nav *Navigator) setBreadcrumbs() {
	nav.breadcrumbs.Clear()

	rootPath := nav.store.RootURL().Path
	{
		rootTitle := nav.store.RootTitle()
		rootTitle = strings.TrimSuffix(rootTitle, "/")
		rootBreadcrumb := sneatv.NewBreadcrumb(rootTitle, func() error {
			nav.goDir(rootPath)
			return nil
		})
		nav.breadcrumbs.Push(rootBreadcrumb)
	}

	relativePath := strings.TrimPrefix(strings.TrimPrefix(nav.current.dir, rootPath), "/")
	if relativePath == "" {
		return
	}
	currentDir := strings.Split(relativePath, "/")
	breadPaths := make([]string, 0, len(currentDir))
	breadPaths = append(breadPaths, rootPath)
	for _, p := range currentDir {
		if p == "" {
			p = "{EMPTY PATH ITEM}"
		}
		breadPaths = append(breadPaths, p)
		breadPath := path.Join(breadPaths...)
		nav.breadcrumbs.Push(sneatv.NewBreadcrumb(p, func() error {
			nav.goDir(breadPath)
			return nil
		}))
	}
}

func (nav *Navigator) getDirData(ctx context.Context) (dirContext *DirContext, err error) {
	dirContext = newDirContext(nav.store, nav.current.dir, nil)
	dirContext.children, err = nav.store.ReadDir(ctx, nav.current.dir)
	if err != nil {
		return nil, err
	}
	// Tree is always sorted by name and files are usually as well
	// So let's sort once here and pass sorted to both Tree and filesPanel.
	dirContext.children = sortDirChildren(dirContext.children)
	return
}

func sortDirChildren(children []os.DirEntry) []os.DirEntry {
	sort.Slice(children, func(i, j int) bool {
		// Directories first
		if children[i].IsDir() && !children[j].IsDir() {
			return true
		} else if !children[i].IsDir() && children[j].IsDir() {
			return false
		}
		// Then sort by name
		return children[i].Name() < children[j].Name()
	})
	return children
}
