package filetug

import (
	"fmt"
	"path"
	"strings"

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
	t.boxed.Draw(screen)
}

func NewTree(nav *Navigator) *Tree {
	tv := tview.NewTreeView()
	t := &Tree{
		nav:      nav,
		TreeView: tv,
		boxed: newBoxed(tv,
			WithLeftPadding(1),
			WithRightBorder(0, 1),
		),
	}
	t.currDirRoot = tview.NewTreeNode("~")
	t.SetRoot(t.currDirRoot)
	t.SetChangedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if dir, ok := ref.(string); ok {
			nav.showDir(dir, node)
		}
	})
	t.SetInputCapture(t.inputCapture)
	return t
}

func (t *Tree) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	tree := t.TreeView
	nav := t.nav
	switch event.Key() {
	case tcell.KeyEnter:
		ref := tree.GetCurrentNode().GetReference()
		if ref != nil {
			dir := ref.(string)
			t.nav.goDir(dir)
			return nil
		}
		return event
	case tcell.KeyUp:
		if tree.GetCurrentNode() == tree.GetRoot() {
			nav.breadcrumbs.TakeFocus(tree)
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
