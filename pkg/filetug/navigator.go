package filetug

import (
	"context"
	"errors"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/filetug/masks"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv/crumbs"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
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

func (nav *Navigator) goRoot() {
	nav.goDirByPath("/")
}

func (nav *Navigator) goHome() {
	nav.goDirByPath("~")
}

func (nav *Navigator) goDirByPath(dirPath string) {
	dirContext := files.NewDirContext(nav.store, dirPath, nil)
	nav.goDir(dirContext)
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

func (nav *Navigator) goDir(dirContext *files.DirContext) {
	if dirContext == nil {
		return
	}
	nav.dirsTree.setCurrentDir(dirContext)
	ctx := context.Background()
	nav.showDir(ctx, nav.dirsTree.rootNode, dirContext, true)
	root := nav.store.RootURL()
	rootValue := root.String()
	nav.saveCurrentDir(rootValue, dirContext.Path())
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
	if statusText == "" {
		return
	}
	nav.app.QueueUpdateDraw(func() {
		node.SetText(cleanPrefix + statusText)
	})
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
		status = getDirStatus(ctx, repo, fullPath)
	} else {
		status = getFileStatus(ctx, repo, fullPath)
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

// TODO: Get rid of package level var saveCurrentDir - see NewNavigator for proper implementation
var saveCurrentDir = ftstate.SaveCurrentDir

func (nav *Navigator) currentDirPath() string {
	currentDir := nav.current.Dir()
	if currentDir == nil {
		return ""
	}
	return currentDir.Path()
}

// showDir updates all panels.
// The `isTreeRootChanged bool` argument is needed do distinguish root dir change from
// the case we simply select the root node in the tree.
func (nav *Navigator) showDir(ctx context.Context, node *tview.TreeNode, dirContext *files.DirContext, isTreeRootChanged bool) {
	if dirContext == nil {
		return
	}
	expandedDir := fsutils.ExpandHome(dirContext.Path())
	if expandedDir != dirContext.Path() {
		dirContext = files.NewDirContext(dirContext.Store(), expandedDir, dirContext.Children())
	}
	currentDirPath := nav.currentDirPath()
	if currentDirPath == expandedDir && !isTreeRootChanged {
		return // TODO: Investigate and document why this happens or fix
	}
	currentChildren := dirContext.Children()
	currentDirContext := files.NewDirContext(dirContext.Store(), expandedDir, currentChildren)
	nav.current.SetDir(currentDirContext)
	if node != nil {
		node.SetReference(dirContext)
	}
	if nav.store != nil && nav.store.RootURL().Scheme == "file" && node != nil {
		name := node.GetText()
		currentDir := nav.current.Dir()
		if currentDir != nil {
			repoRoot := gitutils.GetRepositoryRoot(currentDir.Path())
			var repo *git.Repository
			if repoRoot != "" {
				repo, _ = git.PlainOpen(repoRoot)
			}
			currentPath := currentDir.Path()
			go nav.updateGitStatus(ctx, repo, currentPath, node, name)
		}
	}

	nav.setBreadcrumbs()
	if nav.right != nil {
		nav.right.SetContent(nav.previewer)
	}

	dirPath := expandedDir
	// Start loading data in a goroutine
	go func() {
		dirContext, err := nav.getDirData(ctx, dirPath)
		if nav.app != nil {
			nav.app.QueueUpdateDraw(func() {
				if err != nil {
					nav.showNodeError(node, err)
					return
				}
				nav.onDataLoaded(ctx, node, dirContext, isTreeRootChanged)
			})
		}
	}()
}

func (nav *Navigator) onDataLoaded(ctx context.Context, node *tview.TreeNode, dirContext *files.DirContext, isTreeRootChanged bool) {
	if nav.previewer != nil {
		nav.previewer.PreviewEntry(dirContext)
	}

	//nav.filesPanel.Clear()
	if nav.files != nil {
		nav.files.table.SetSelectable(true, false)

		dirRecords := NewFileRows(dirContext)
		nav.files.SetRows(dirRecords, node != nil && node != nav.dirsTree.rootNode)
	}

	if isTreeRootChanged && node != nil && nav.dirsTree != nil {
		nav.dirsTree.setDirContext(ctx, node, dirContext)
	}
	if nav.files != nil {
		nav.files.updateGitStatuses(ctx, dirContext)
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

func (nav *Navigator) getDirData(ctx context.Context, dirPath string) (dirContext *files.DirContext, err error) {
	if nav.store == nil {
		return nil, errors.New("store not set")
	}
	dirContext = files.NewDirContext(nav.store, dirPath, nil)
	var children []os.DirEntry
	children, err = nav.store.ReadDir(ctx, dirPath)
	if err != nil {
		return nil, err
	}
	// Tree is always sorted by name and files are usually as well
	// So let's sort once here and pass sorted to both Tree and filesPanel.
	children = sortDirChildren(children)
	//time.Sleep(time.Millisecond * 2000)
	dirContext.SetChildren(children)
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
	if nav.right != nil {
		nav.right.SetContent(nav.masks)
	}
}

func (nav *Navigator) showNewPanel() {
	if nav.newPanel != nil {
		nav.newPanel.Show()
	}
}
