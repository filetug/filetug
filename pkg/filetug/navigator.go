package filetug

import (
	"context"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/ftpfile"
	"github.com/filetug/filetug/pkg/files/httpfile"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/filetug/masks"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

type Navigator struct {
	app             *tview.Application                          // Should we get rid of it?
	queueUpdateDraw func(f func())                              // replaced with mock in tests
	setAppFocus     func(p tview.Primitive)                     // replaced with mock in tests
	setAppRoot      func(root tview.Primitive, fullscreen bool) // replaced with mock in tests
	stopApp         func()

	o navigatorOptions

	store files.Store

	breadcrumbs *crumbs.Breadcrumbs

	*tview.Flex
	main *tview.Flex

	current     current
	activeCol   int
	proportions []int

	filesSelectionChangedFunc func(row, column int)

	favoritesFocusFunc func()

	previewerFocusFunc func()
	previewerBlurFunc  func()

	left  *Container
	right *Container

	dirsTree  *Tree
	favorites *favorites
	masks     *masks.Panel
	newPanel  *NewPanel

	files *filesPanel

	dirSummary *dirSummary

	previewer *previewerPanel

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
	nav.setAppFocus(nav.dirsTree.tv)
}

func (nav *Navigator) SetFocusToContainer(index int) {
	switch index {
	case nav.left.index:
		nav.setAppFocus(nav.left.Flex)
	case nav.right.index:
		nav.setAppFocus(nav.right.Flex)
	case 1:
		nav.setAppFocus(nav.files.Boxed)
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

var getState = ftstate.GetState

func NewNavigator(app *tview.Application, options ...NavigatorOption) *Navigator {
	defaultStore := osfile.NewStore("/")
	rootBreadcrumb := crumbs.NewBreadcrumb("FileTug: ", func() error {
		return nil
	})
	rootBreadcrumb.SetColor(tcell.ColorWhiteSmoke)
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	nav := &Navigator{
		app: app,
		queueUpdateDraw: func(f func()) {
			app.QueueUpdateDraw(f)
		},
		setAppFocus: func(p tview.Primitive) {
			app.SetFocus(p)
		},
		setAppRoot: func(root tview.Primitive, fullscreen bool) {
			app.SetRoot(root, fullscreen)
		},
		stopApp: app.Stop,
		store:   defaultStore,
		breadcrumbs: crumbs.NewBreadcrumbs(
			rootBreadcrumb,
			crumbs.WithSeparator("/"),
			crumbs.WithSeparatorStartIndex(1),
		),
		Flex:           flex,
		main:           tview.NewFlex(),
		proportions:    make([]int, 3),
		gitStatusCache: make(map[string]*gitutils.RepoStatus),
	}
	nav.bottom = newBottom(nav)
	nav.right = NewContainer(2, nav)
	nav.favorites = newFavorites(nav)
	nav.dirsTree = NewTree(nav)
	nav.newPanel = NewNewPanel(nav)
	nav.AddItem(nav.breadcrumbs, 1, 0, false)

	copy(nav.proportions, defaultProportions)

	nav.files = newFiles(nav)
	nav.previewer = newPreviewerPanel(nav)
	nav.dirSummary = newDirSummary(nav)
	nav.right.SetContent(nav.dirSummary)

	for _, option := range options {
		option(&nav.o)
	}

	createLeft(nav)

	nav.AddItem(nav.main, 0, 1, true)

	nav.AddItem(nav.bottom, 1, 0, false)

	nav.createColumns()

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
				return nav
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

func (nav *Navigator) getCurrentBrowser() browser {
	switch nav.activeCol {
	case 0:
		return nav.dirsTree
	case 1:
		return nav.files
	}
	return nil
}

func (nav *Navigator) inputCapture(event *tcell.EventKey) *tcell.EventKey {
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
				nav.favorites.ShowFavorites()
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
				nav.goDir("/")
				return nil
			case '~', 'h', 'H':
				nav.goDir("~")
				return nil
			case 'x', 'X':
				nav.stopApp()
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
	rootValue := root.String()
	saveCurrentDir(rootValue, dir)
}

func (nav *Navigator) updateGitStatus(ctx context.Context, repo *git.Repository, fullPath string, node *tview.TreeNode, prefix string) {
	cleanPrefix := stripGitStatusPrefix(prefix)
	if node == nil {
		return
	}
	status := nav.getGitStatus(ctx, repo, fullPath, true)
	if status == nil {
		return
	}
	statusText := nav.gitStatusText(status, fullPath, true)
	if nav.app != nil {
		nav.queueUpdateDraw(func() {
			node.SetText(cleanPrefix + statusText)
		})
	} else {
		node.SetText(cleanPrefix + statusText)
	}
}

const gitStatusSeparator = "[gray]┆[-]"

func stripGitStatusPrefix(text string) string {
	separatorIndex := strings.Index(text, gitStatusSeparator)
	if separatorIndex == -1 {
		return text
	}
	return text[:separatorIndex]
}

func (nav *Navigator) getGitStatus(ctx context.Context, repo *git.Repository, fullPath string, isDir bool) *gitutils.RepoStatus {
	nav.gitStatusCacheMu.RLock()
	cachedStatus, ok := nav.gitStatusCache[fullPath]
	nav.gitStatusCacheMu.RUnlock()
	if ok {
		return cachedStatus
	}

	if repo == nil {
		repoRoot := gitutils.GetRepositoryRoot(fullPath)
		if repoRoot == "" {
			return nil
		}

		var err error
		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return nil
		}
	}

	var status *gitutils.RepoStatus
	if isDir {
		status = gitutils.GetDirStatus(ctx, repo, fullPath)
	} else {
		status = gitutils.GetFileStatus(ctx, repo, fullPath)
	}
	if status == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	nav.gitStatusCacheMu.Lock()
	nav.gitStatusCache[fullPath] = status
	nav.gitStatusCacheMu.Unlock()
	return status
}

func (nav *Navigator) gitStatusText(status *gitutils.RepoStatus, fullPath string, isDir bool) string {
	if status == nil {
		return ""
	}

	hasChanges := status.FilesChanged > 0 || status.Insertions > 0 || status.Deletions > 0
	isRepoRoot := false
	if isDir {
		repoRoot := gitutils.GetRepositoryRoot(fullPath)
		isRepoRoot = repoRoot != "" && (fullPath == repoRoot || fullPath == repoRoot+"/")
	}
	if hasChanges || isRepoRoot {
		return status.String()
	}
	return ""
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
	if nav.store.RootURL().Scheme == "file" && node != nil {
		name := node.GetText()
		repoRoot := gitutils.GetRepositoryRoot(nav.current.dir)
		var repo *git.Repository
		if repoRoot != "" {
			repo, _ = git.PlainOpen(repoRoot)
		}
		go nav.updateGitStatus(ctx, repo, nav.current.dir, node, name)
	}

	nav.setBreadcrumbs()
	nav.right.SetContent(nav.dirSummary)
	nav.previewer.textView.SetText("").SetTextColor(tcell.ColorWhiteSmoke)

	go func() {
		dirContext, err := nav.getDirData(ctx)
		nav.queueUpdateDraw(func() {
			if err != nil {
				nav.showNodeError(node, err)
				return
			}
			nav.onDataLoaded(ctx, node, dirContext, isTreeRootChanged)
		})
	}()
}

func (nav *Navigator) onDataLoaded(ctx context.Context, node *tview.TreeNode, dirContext *DirContext, isTreeRootChanged bool) {
	nav.dirSummary.SetDir(dirContext)

	//nav.filesPanel.Clear()
	nav.files.table.SetSelectable(true, false)

	dirRecords := NewFileRows(dirContext)
	nav.files.SetRows(dirRecords, node != nav.dirsTree.rootNode)

	if isTreeRootChanged {
		nav.dirsTree.setDirContext(ctx, node, dirContext)
	}
	nav.files.updateGitStatuses(ctx, dirContext)
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
	if rootPath == "" {
		rootPath = "/"
	}
	{
		rootTitle := nav.store.RootTitle()
		rootTitle = strings.TrimSuffix(rootTitle, "/")
		rootBreadcrumb := crumbs.NewBreadcrumb(rootTitle, func() error {
			nav.goDir(rootPath)
			return nil
		})
		nav.breadcrumbs.Push(rootBreadcrumb)
	}

	trimmedDir := strings.TrimPrefix(nav.current.dir, rootPath)
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
			nav.goDir(breadPath)
			return nil
		})
		nav.breadcrumbs.Push(breadcrumb)
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

func (nav *Navigator) showMasks() {
	if nav.masks == nil {
		nav.masks = masks.NewPanel()
	}
	nav.right.SetContent(nav.masks)
}

func (nav *Navigator) showNewPanel() {
	nav.newPanel.Show()
}
