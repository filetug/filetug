package filetug

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/datatug/filetug/pkg/sneatv"
	"github.com/datatug/filetug/pkg/sticky"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Navigator struct {
	app *tview.Application
	o   navigatorOptions

	breadcrumbs *sneatv.Breadcrumbs

	*tview.Flex
	main *tview.Flex

	currentDir  string
	activeCol   int
	proportions []int

	filesFocusFunc            func()
	filesBlurFunc             func()
	filesSelectionChangedFunc func(row, column int)

	favoritesFocusFunc func()
	favoritesBlurFunc  func()

	dirsFocusFunc func()
	dirsBlurFunc  func()

	previewerFocusFunc func()
	previewerBlurFunc  func()

	left      *left
	dirsTree  *Tree
	favorites *favorites

	files *sticky.Table

	previewer *previewer
}

func (nav *Navigator) SetFocus() {
	nav.app.SetFocus(nav.dirsTree.TreeView)
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
		Flex:        tview.NewFlex().SetDirection(tview.FlexRow),
		main:        tview.NewFlex(),
		favorites:   newFavorites(),
		proportions: make([]int, 3),
	}
	nav.dirsTree = NewTree(nav)
	nav.AddItem(nav.breadcrumbs, 1, 0, false)

	copy(nav.proportions, defaultProportions)

	nav.files = newFiles(nav)
	nav.previewer = newPreviewer(nav)

	for _, option := range options {
		option(&nav.o)
	}

	createLeft(nav)

	nav.AddItem(nav.main, 0, 1, true)

	nav.createColumns()

	nav.goDir("~")

	return nav
}

var defaultProportions = []int{6, 10, 8}

func (nav *Navigator) createColumns() {

	nav.main.Clear()
	nav.main.AddItem(nav.left, 0, nav.proportions[0], true)
	nav.main.AddItem(nav.files, 0, nav.proportions[1], true)
	nav.main.AddItem(nav.previewer, 0, nav.proportions[2], true)

	nav.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Modifiers()&tcell.ModAlt != 0 {
			if event.Key() == tcell.KeyRune {
				switch r := event.Rune(); r {
				case '0':
					copy(nav.proportions, defaultProportions)
					nav.createColumns()
				case '+', '=':
					switch nav.activeCol {
					case 0:
						nav.proportions[0] += 2
						nav.proportions[1] -= 1
						nav.proportions[2] -= 1
					case 1:
						nav.proportions[0] -= 1
						nav.proportions[1] += 2
						nav.proportions[2] -= 1
					case 2:
						nav.proportions[0] -= 1
						nav.proportions[1] -= 1
						nav.proportions[2] += 2
					default:
						return event
					}
					nav.createColumns()
					return nil
				case '-', '_':
					switch nav.activeCol {
					case 0:
						nav.proportions[0] -= 2
						nav.proportions[1] += 1
						nav.proportions[2] += 1
					case 1:
						nav.proportions[0] += 1
						nav.proportions[1] -= 2
						nav.proportions[2] += 1
					case 2:
						nav.proportions[0] += 1
						nav.proportions[1] += 1
						nav.proportions[2] -= 2
					default:
						return event
					}
					nav.createColumns()
					return nil
				case '/', 'r', 'R':
					nav.goDir("/")
					return nil
				case '~', 'h', 'H':
					nav.goDir("~")
					return nil
				default:
					return event
				}
			}
		}
		return event
	})
}

func (nav *Navigator) goDir(dir string) {

	nav.favorites.SetCurrentNode(nil)
	nav.showDir(dir, nil)
}

func (nav *Navigator) showDir(dir string, selectedNode *tview.TreeNode) {

	var parentNode *tview.TreeNode
	var nodePath string

	isTreeDirChanges := selectedNode == nil

	if isTreeDirChanges {
		nav.favorites.SetCurrentNode(nil)
		nav.dirsTree.currDirRoot.ClearChildren()
		parentNode = nav.dirsTree.currDirRoot
	} else {
		nav.dirsTree.selectedDirNode = selectedNode
		parentNode = selectedNode
	}

	if strings.HasPrefix(dir, "~") || strings.HasPrefix(dir, "/") {
		nodePath = dir[:1]
		nav.dirsTree.currDirRoot.SetText(nodePath).SetReference(nodePath)
	}

	dirRelPath := strings.TrimPrefix(strings.TrimPrefix(dir, "~"), "/")

	if dirRelPath != "" {
		parents := strings.Split(dirRelPath, "/")
		for _, p := range parents {
			if nodePath == "/" {
				nodePath += p
			} else {
				nodePath = nodePath + "/" + p
			}
			n := tview.NewTreeNode("📁" + p).SetReference(nodePath)
			if isTreeDirChanges {
				parentNode.AddChild(n)
				parentNode = n
			}
		}
	}

	if isTreeDirChanges {
		nav.dirsTree.selectedDirNode = parentNode
	}
	nav.currentDir = fsutils.ExpandHome(nodePath)

	nav.breadcrumbs.Clear()

	for _, p := range strings.Split(nav.currentDir, "/") {
		if p == "" {
			continue
		}
		nav.breadcrumbs.Push(sneatv.NewBreadcrumb(p, nil))
	}

	children, err := os.ReadDir(nav.currentDir)
	if err != nil {
		parentNode.ClearChildren()
		parentNode.AddChild(tview.NewTreeNode(fmt.Sprintf("Error for %s: %s", nav.currentDir, err.Error())))
		return
	}

	if isTreeDirChanges {
		nav.files.SetTitle(fmt.Sprintf(" Files: %s ", dir))
	} else {
		nav.files.SetTitle(dir)
	}
	nav.files.Clear()
	//nav.files.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(Style.TableHeaderColor).SetExpansion(1))
	//nav.files.SetCell(0, 1, tview.NewTableCell("Size").SetTextColor(Style.TableHeaderColor).SetAlign(tview.AlignRight))
	//nav.files.SetCell(0, 2, tview.NewTableCell("Modified").SetTextColor(Style.TableHeaderColor).SetAlign(tview.AlignRight))
	//nav.files.SetFixed(1, 0)
	nav.files.SetSelectable(true, false)
	//nav.files.Select(1, 0)

	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir() && !children[j].IsDir() {
			return true
		} else if !children[i].IsDir() && children[j].IsDir() {
			return false
		}
		return children[i].Name() < children[j].Name()
	})

	nav.files.SetRecords(fsRecords{nodePath: nodePath, dirEntries: children})

	for _, child := range children {
		name := child.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if child.IsDir() && isTreeDirChanges {
			n := tview.NewTreeNode("📁" + name).SetReference(path.Join(nodePath, name))
			parentNode.AddChild(n)
		}
	}
	if isTreeDirChanges {
		nav.dirsTree.SetCurrentNode(parentNode)
	}
	//nav.app.SetFocus(nav.dirsTree)
	//nav.app.QueueUpdate(func() {
	//})
}
