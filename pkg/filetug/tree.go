package filetug

import (
	"github.com/rivo/tview"
)

type Tree struct {
	*tview.TreeView
	currDirRoot *tview.TreeNode
}

func (t *Tree) GetBox() *tview.Box {
	return t.Box
}

func NewTree() *Tree {
	t := &Tree{
		TreeView: tview.NewTreeView(),
	}

	t.currDirRoot = tview.NewTreeNode("~")
	t.SetRoot(t.currDirRoot)

	return t
}
