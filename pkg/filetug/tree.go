package filetug

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/datatug/filetug/pkg/filetug/ftstate"
	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/datatug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const dirEmoji = "📁"

type Tree struct {
	boxed *sneatv.Boxed
	*tview.TreeView
	nav             *Navigator
	rootNode        *tview.TreeNode
	search          string
	loadingProgress int
	queueUpdateDraw func(f func()) *tview.Application
}

func (t *Tree) onStoreChange() {
	t.loadingProgress = 0
	t.rootNode.ClearChildren()
	rootPath := t.nav.store.RootURL().Path
	if rootPath == "" {
		rootPath = "/"
	}
	t.rootNode.SetText(rootPath)
	loadingNode := tview.NewTreeNode(" Loading...")
	loadingNode.SetColor(tcell.ColorGray)
	t.rootNode.AddChild(loadingNode)
	go func() {
		t.doLoadingAnimation(loadingNode)
	}()
}

var spinner = []rune("▏▎▍▌▋▊▉█")

func (t *Tree) doLoadingAnimation(loading *tview.TreeNode) {
	time.Sleep(50 * time.Millisecond)
	if children := t.rootNode.GetChildren(); len(children) == 1 && children[0] == loading {
		q, r := t.loadingProgress/len(spinner), t.loadingProgress%len(spinner)
		progressBar := strings.Repeat("█", q) + string(spinner[r])
		t.queueUpdateDraw(func() {
			loading.SetText(" Loading... " + progressBar)
		})
		t.loadingProgress += 1
		t.doLoadingAnimation(loading)
	}
}

func (t *Tree) Draw(screen tcell.Screen) {
	root := t.GetRoot()
	text := root.GetText()
	if strings.HasSuffix(text, " ") {
		_, _, width, _ := t.GetInnerRect()
		if width > len(text) {
			text += strings.Repeat(" ", width-len(text))
			root.SetText(text)
		}
	}
	t.boxed.Draw(screen)
}

func NewTree(nav *Navigator) *Tree {
	tv := tview.NewTreeView()
	t := &Tree{nav: nav, TreeView: tv,
		boxed: sneatv.NewBoxed(tv, sneatv.WithRightBorder(0, 1)),
	}
	t.rootNode = tview.NewTreeNode("~")
	t.SetRoot(t.rootNode)
	t.SetChangedFunc(t.changed)
	t.SetInputCapture(t.inputCapture)
	t.SetFocusFunc(t.focus)
	t.SetBlurFunc(t.blur)
	t.queueUpdateDraw = nav.app.QueueUpdateDraw
	return t
}

func (t *Tree) changed(node *tview.TreeNode) {
	ref := node.GetReference()

	if dir, ok := ref.(string); ok {
		var ctx context.Context
		ctx, t.nav.cancel = context.WithCancel(context.Background())
		t.nav.showDir(ctx, node, dir, false)
		ftstate.SaveSelectedTreeDir(dir)
	}
}

func (t *Tree) setError(node *tview.TreeNode, err error) {
	//panic(err)
	node.ClearChildren()
	node.SetColor(tcell.ColorOrangeRed)
	nodePath := getNodePath(node)
	_, name := path.Split(nodePath)

	text := dirEmoji + fmt.Sprintf("%s: %v", name, err)
	node.SetText(text)
	//node.AddChild(tview.NewTreeNode(err.Error()))
}

func getNodePath(node *tview.TreeNode) string {
	return node.GetReference().(string)
}

func (t *Tree) focus() {
	t.nav.left.SetBorderColor(sneatv.CurrentTheme.FocusedBorderColor)
	t.nav.activeCol = 0
	t.nav.right.SetContent(t.nav.dirSummary)
	t.nav.dirSummary.Blur()
	t.nav.right.Blur()
	t.nav.files.blur()
	currentNode := t.GetCurrentNode()
	if currentNode == nil {
		currentNode = t.GetRoot()
		t.SetCurrentNode(currentNode)
	}
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(sneatv.CurrentTheme.FocusedSelectedTextStyle)
	}
	t.SetGraphicsColor(tcell.ColorWhite)
}

func (t *Tree) blur() {
	t.nav.left.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	t.SetGraphicsColor(sneatv.CurrentTheme.BlurredGraphicsColor)
	currentNode := t.GetCurrentNode()
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(sneatv.CurrentTheme.BlurredSelectedTextStyle)
	}
}

func (t *Tree) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	tree := t.TreeView
	nav := t.nav
	switch event.Key() {
	case tcell.KeyRight:
		t.nav.app.SetFocus(t.nav.files)
		return nil
	case tcell.KeyLeft:
		switch ref := tree.GetCurrentNode().GetReference().(type) {
		case string:
			parentDir, _ := path.Split(ref)
			t.nav.goDir(parentDir)
			return nil
		}
		return event
	case tcell.KeyEnter:
		switch ref := tree.GetCurrentNode().GetReference().(type) {
		case string:
			if ref != "/" {
				ref = strings.TrimSuffix(ref, "/")
			}
			if t.GetCurrentNode() == t.GetRoot() {
				ref, _ = path.Split(fsutils.ExpandHome(ref))
			}
			t.nav.goDir(ref)
			return nil
		}
		return event
	case tcell.KeyUp:
		if tree.GetCurrentNode() == tree.GetRoot() {
			nav.breadcrumbs.TakeFocus(tree)
			nav.app.SetFocus(nav.breadcrumbs)
			return nil
		}
		return event
	case tcell.KeyBackspace:
		t.SetSearch(t.search[:len(t.search)-1])
		return nil
	case tcell.KeyEscape:
		t.SetSearch("")
		return nil
	case tcell.KeyRune:
		s := string(event.Rune())
		if t.search == "" && s == " " {
			return event
		}
		t.SetSearch(t.search + strings.ToLower(s))
		return nil
	default:
		return event
	}
}

func (t *Tree) SetSearch(pattern string) {
	t.search = pattern
	if pattern == "" {
		t.search = ""
		t.SetTitle("")
	} else {
		t.SetTitle(fmt.Sprintf("Find: %s", t.search))
	}
	search := &search{
		pattern: t.search,
	}
	highlightTreeNodes(t.GetRoot(), search)
	if search.firstPrefixed != nil {
		t.SetCurrentNode(search.firstPrefixed)
	} else if search.firstContains != nil {
		t.SetCurrentNode(search.firstContains)
	} else if len(t.search) > 0 {
		t.SetSearch(t.search[:len(t.search)-1])
	}
}

type search struct {
	pattern       string
	found         []string
	firstContains *tview.TreeNode
	firstPrefixed *tview.TreeNode
}

func highlightTreeNodes(n *tview.TreeNode, search *search) {
	r := n.GetReference()
	if s, ok := r.(string); ok {
		if _, name := path.Split(s); strings.Contains(strings.ToLower(name), search.pattern) {
			i := strings.Index(strings.ToLower(name), search.pattern)
			ss := name[i : i+len(search.pattern)]
			formatted := fmt.Sprintf("[black:lightgreen]%s[-:-]", ss)
			text := strings.ReplaceAll(name, ss, formatted)
			n.SetText(text)
			search.found = append(search.found, text)
			if search.firstContains == nil {
				search.firstContains = n
			}
			if search.firstPrefixed == nil && strings.HasPrefix(strings.ToLower(name), search.pattern) {
				search.firstPrefixed = n
			}
		} else {
			n.SetText(name)
		}
	}
	for _, child := range n.GetChildren() {
		highlightTreeNodes(child, search)
	}
}

func (t *Tree) setCurrentDir(dir string) (nodePath string) {
	t.SetSearch("")
	t.rootNode.ClearChildren()

	root := t.nav.store.RootURL()
	if root.Path == "" {
		root.Path = "/"
	}

	var text string
	if dir == root.Path {
		if dir == "/" {
			text = "/"
		} else {
			text = strings.TrimSuffix(root.Path, "/")
		}
	} else {
		nodePath = strings.TrimPrefix(dir, root.Path)
		text = ".."
	}

	t.rootNode.SetText(text)
	t.rootNode.SetReference(nodePath).SetColor(tcell.ColorWhite)

	return
}

func (t *Tree) setDirContext(ctx context.Context, node *tview.TreeNode, dirContext *DirContext) {
	node.ClearChildren()
	for _, child := range dirContext.children {
		name := child.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if child.IsDir() {
			childPath := path.Join(dirContext.Path, name)
			emoji := dirEmoji
			switch strings.ToLower(name) {
			case "library":
				emoji = "📚"
			case "users":
				emoji = "👥"
			case "applications":
				emoji = "🈸"
			case "music":
				emoji = "🎹"
			case "movies":
				emoji = "📺"
			case "pictures":
				emoji = "🖼️"
			case "desktop":
				emoji = "🖥️"
			case "datatug":
				emoji = "🛥️"
			case "documents":
				emoji = "🗃"
			case "public":
				emoji = "📢"
			case "temp":
				emoji = "ƒ⏳"
			case "system":
				emoji = "🧠"
			case "bin", "sbin":
				emoji = "🚀"
			case "private":
				emoji = "🔒"
			}
			prefix := emoji + name
			n := tview.NewTreeNode(prefix).SetReference(childPath)
			node.AddChild(n)

			fullPath := fsutils.ExpandHome(childPath)
			go t.nav.updateGitStatus(ctx, fullPath, n, prefix+" ")
		}
	}
}
