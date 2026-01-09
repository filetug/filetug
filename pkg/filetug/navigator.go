package filetug

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Navigator struct {
	app *tview.Application
	o   navigatorOptions

	*tview.Flex

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

	leftFocusFunc func()
	leftBlurFunc  func()

	previewerFocusFunc func()
	previewerBlurFunc  func()

	left      *tview.Flex
	dirs      *Tree
	favorites *favorites

	files *tview.Table

	previewer *previewer
}

func (nav *Navigator) SetFocus() {
	nav.app.SetFocus(nav.dirs.TreeView)
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
		app:         app,
		dirs:        NewTree(),
		favorites:   newFavorites(),
		proportions: make([]int, 3),
	}
	copy(nav.proportions, defaultProportions)

	nav.files = newFiles(nav)
	nav.previewer = newPreviewer(nav)

	for _, option := range options {
		option(&nav.o)
	}

	createLeft(nav)

	nav.createColumns()

	nav.goDir("~")

	return nav
}

var defaultProportions = []int{6, 10, 8}

func (nav *Navigator) createColumns() {
	nav.Flex = tview.NewFlex()
	nav.AddItem(nav.left, 0, nav.proportions[0], true)
	nav.AddItem(nav.files, 0, nav.proportions[1], true)
	nav.AddItem(nav.previewer, 0, nav.proportions[2], true)
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

	t := nav.dirs
	t.currDirRoot.ClearChildren()

	parentNode := t.currDirRoot

	var nodePath string

	if strings.HasPrefix(dir, "~") || strings.HasPrefix(dir, "/") {
		nodePath = dir[:1]
		t.currDirRoot.SetText(nodePath).SetReference(nodePath)
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
			parentNode.AddChild(n)
			parentNode = n
		}
	}

	nav.currentDir = fsutils.ExpandHome(nodePath)
	children, err := os.ReadDir(nav.currentDir)
	if err != nil {
		parentNode.AddChild(tview.NewTreeNode(fmt.Sprintf("Error for %s: %s", nav.currentDir, err.Error())))
		return
	}
	fileIndex := 0

	nav.files.SetTitle(fmt.Sprintf(" Files: %s ", dir))
	nav.files.Clear()
	nav.files.SetCell(0, 0, tview.NewTableCell("File name").SetTextColor(Style.TableHeaderColor).SetExpansion(1))
	nav.files.SetCell(0, 1, tview.NewTableCell("Size").SetTextColor(Style.TableHeaderColor).SetAlign(tview.AlignRight))
	nav.files.SetCell(0, 2, tview.NewTableCell("Modified").SetTextColor(Style.TableHeaderColor).SetAlign(tview.AlignRight))
	nav.files.SetFixed(1, 0)
	nav.files.SetSelectable(true, false)
	//nav.files.Select(1, 0)

	fileIndex++
	for _, child := range children {
		name := child.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if child.IsDir() {
			n := tview.NewTreeNode("📁" + name).SetReference(path.Join(nodePath, name))
			parentNode.AddChild(n)
		} else {
			tdName := tview.NewTableCell(" " + name)
			color := GetColorByFileName(name)
			tdName.SetTextColor(color)
			nav.files.SetCell(fileIndex, 0, tdName)
			if fi, err := child.Info(); err == nil {
				td := tview.NewTableCell(strconv.FormatInt(fi.Size(), 10)).SetAlign(tview.AlignRight).SetTextColor(color)
				nav.files.SetCell(fileIndex, 1, td)
				modTime := fi.ModTime()
				var modStr string
				if modTime.After(time.Now().Add(24 * time.Hour)) {
					modStr = modTime.Format("15:04:05")
				} else {
					modStr = modTime.Format("2006-01-02")
				}
				td = tview.NewTableCell(modStr).SetAlign(tview.AlignRight).SetTextColor(color)
				nav.files.SetCell(fileIndex, 2, td)
			}
			fileIndex++
		}
	}
	t.SetCurrentNode(parentNode)
	nav.app.SetFocus(t)
}
