package filetug

import (
	"fmt"
	"path"
	"strings"

	"github.com/datatug/filetug/pkg/fsutils"
	"github.com/datatug/filetug/pkg/ftstate"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Tree struct {
	boxed *boxed
	*tview.TreeView
	nav             *Navigator
	currDirRoot     *tview.TreeNode
	search          string
	selectedDirNode *tview.TreeNode
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
	t := &Tree{
		nav:      nav,
		TreeView: tv,
		boxed: newBoxed(tv,
			//WithLeftPadding(0),
			WithRightBorder(0, 1),
		),
	}
	t.currDirRoot = tview.NewTreeNode("~")
	t.SetRoot(t.currDirRoot)
	t.SetChangedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()

		if dir, ok := ref.(string); ok {
			_, name := path.Split(dir)
			ftstate.SaveSelectedTreeDir(name)
			nav.showDir(dir, node)
		}
	})
	t.SetInputCapture(t.inputCapture)
	t.SetFocusFunc(t.focus)
	t.SetBlurFunc(t.blur)
	return t
}

func (t *Tree) focus() {
	t.nav.left.SetBorderColor(theme.FocusedBorderColor)
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
		currentNode.SetSelectedTextStyle(theme.FocusedSelectedTextStyle)
	}
	t.SetGraphicsColor(tcell.ColorWhite)
}

func (t *Tree) blur() {
	t.nav.left.SetBorderColor(theme.BlurredBorderColor)
	t.SetGraphicsColor(theme.BlurredGraphicsColor)
	currentNode := t.GetCurrentNode()
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(theme.BlurredSelectedTextStyle)
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
