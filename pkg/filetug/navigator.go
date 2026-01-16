package filetug

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

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

	files *files

	dirSummary *dirSummary

	previewer *previewer

	bottom *bottom

	gitStatusCache   map[string]*gitutils.RepoStatus
	gitStatusCacheMu sync.RWMutex
	gitCancel        context.CancelFunc
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
		app: app,
		breadcrumbs: sneatv.NewBreadcrumbs(
			sneatv.NewBreadcrumb("FileTug: ", func() error {
				return nil
			}).SetColor(tcell.ColorWhiteSmoke),
			sneatv.WithSeparator("/"),
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
		if state.CurrentDir == "" {
			state.CurrentDir = "~"
		}
		nav.goDir(state.CurrentDir)
		if stateErr == nil {
			if state.CurrentFileName != "" {
				nav.files.SetCurrentFile(state.CurrentFileName)
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
	nav.dirsTree.SetSearch("")
	nav.showDir(dir, nil)
	saveCurrentDir(dir)
}

func (nav *Navigator) updateGitStatus(ctx context.Context, fullPath string, node *tview.TreeNode, prefix string) {
	nav.gitStatusCacheMu.RLock()
	cachedStatus, ok := nav.gitStatusCache[fullPath]
	nav.gitStatusCacheMu.RUnlock()

	if ok {
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

func (nav *Navigator) showDir(dir string, selectedNode *tview.TreeNode) {
	nav.current.dir = dir

	isTreeDirChanges := selectedNode == nil

	//if isTreeDirChanges {
	//	if nav.gitCancel != nil {
	//		nav.gitCancel()
	//	}
	//}

	var ctx context.Context
	ctx, nav.gitCancel = context.WithCancel(context.Background())

	var parentNode *tview.TreeNode
	var nodePath string

	if isTreeDirChanges {
		nav.dirsTree.currDirRoot.ClearChildren()
		parentNode = nav.dirsTree.currDirRoot
	} else {
		nav.dirsTree.selectedDirNode = selectedNode
		parentNode = selectedNode
	}

	if strings.HasPrefix(dir, "~") || strings.HasPrefix(dir, "/") {
		nodePath = dir
		if isTreeDirChanges {
			fullPath := fsutils.ExpandHome(nodePath)
			rootNode := nav.dirsTree.currDirRoot
			switch dir {
			case "/":
				rootNode.SetText(dir + strings.Repeat(" ", 10))
			default:
				rootNode.SetText("..")
			}

			rootNode.SetReference(nodePath).SetColor(tcell.ColorWhite)
			go nav.updateGitStatus(ctx, fullPath, nav.dirsTree.currDirRoot, nodePath)
		}
	}

	//dirRelPath := strings.TrimPrefix(strings.TrimPrefix(dir, "~"), "/")
	//
	//if dirRelPath != "" {
	//	parents := strings.Split(dirRelPath, "/")
	//	for _, p := range parents {
	//		if ParentPath == "/" {
	//			ParentPath += p
	//		} else {
	//			ParentPath = ParentPath + "/" + p
	//		}
	//		if isTreeDirChanges {
	//			fullPath := fsutils.ExpandHome(ParentPath)
	//			prefix := "📁" + p
	//			n := tview.NewTreeNode(prefix).SetReference(ParentPath)
	//			go nav.updateGitStatus(ctx, fullPath, n, prefix)
	//			parentNode.AddChild(n)
	//			parentNode = n
	//		}
	//	}
	//}

	if isTreeDirChanges {
		nav.dirsTree.selectedDirNode = parentNode
	}
	nav.current.dir = fsutils.ExpandHome(nodePath)

	nav.breadcrumbs.Clear()

	currentDir := strings.Split(nav.current.dir, "/")
	breadPaths := make([]string, 0, len(currentDir))
	for _, p := range currentDir[1:] {
		if p == "" {
			continue
		}
		breadPaths = append(breadPaths, p)
		breadPath := "/" + path.Join(breadPaths...)
		nav.breadcrumbs.Push(sneatv.NewBreadcrumb(p, func() error {
			nav.goDir(breadPath)
			return nil
		}))
	}

	children, err := os.ReadDir(nav.current.dir)
	dirContext := newDirContext(nav.current.dir, children)

	nav.dirSummary.SetDir(dirContext)
	nav.right.SetContent(nav.dirSummary)

	if err != nil {
		parentNode.ClearChildren()
		parentNode.SetColor(tcell.ColorOrangeRed)
		dirEntry := DirEntry{
			Path: nodePath,
			//DirEntry: ,
		}
		dirRecords := NewFileRows(dirEntry, nil)
		nav.files.SetRows(dirRecords)
		nav.previewer.textView.SetText(err.Error()).SetWrap(true).SetTextColor(tcell.ColorOrangeRed)
		return
	}
	nav.previewer.textView.SetText("").SetTextColor(tcell.ColorWhiteSmoke)

	if isTreeDirChanges {
		nav.files.SetTitle(fmt.Sprintf(" Files: %s ", dir))
	} else {
		nav.files.SetTitle(dir)
	}
	//nav.files.Clear()
	nav.files.table.SetSelectable(true, false)

	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir() && !children[j].IsDir() {
			return true
		} else if !children[i].IsDir() && children[j].IsDir() {
			return false
		}
		return children[i].Name() < children[j].Name()
	})

	dirEntry := DirEntry{
		Path: nodePath,
	}
	dirRecords := NewFileRows(dirEntry, children)
	nav.files.SetRows(dirRecords)

	if isTreeDirChanges {
		for _, child := range children {
			name := child.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if child.IsDir() {
				childPath := path.Join(nodePath, name)
				prefix := "📁" + name
				n := tview.NewTreeNode(prefix).SetReference(childPath)
				parentNode.AddChild(n)

				fullPath := fsutils.ExpandHome(childPath)
				go nav.updateGitStatus(ctx, fullPath, n, prefix+" ")
			}
		}
		nav.dirsTree.SetCurrentNode(parentNode)
	}
}
