package filetug

import (
	"os"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/rivo/tview"
)

// Tree represents the directory tree view component
type Tree struct {
	*sneatv.Boxed
	tv              *tview.TreeView
	nav             *Navigator
	rootNode        *tview.TreeNode
	searchPattern   string
	loadingProgress int
}

// treeDirEntry is a wrapper for os.DirEntry used in testing
type treeDirEntry struct {
	os.DirEntry
	name  string
	isDir bool
}

func (e *treeDirEntry) Name() string {
	return e.name
}

func (e *treeDirEntry) IsDir() bool {
	return e.isDir
}

// NewTree creates a new Tree instance
func NewTree(nav *Navigator) *Tree {
	tv := tview.NewTreeView()
	rightBorder := sneatv.WithRightBorder(0, 1)
	t := &Tree{nav: nav, tv: tv,
		Boxed: sneatv.NewBoxed(tv, rightBorder),
	}
	t.rootNode = tview.NewTreeNode("~")
	tv.SetRoot(t.rootNode)
	tv.SetChangedFunc(t.changed)
	tv.SetInputCapture(t.inputCapture)
	tv.SetFocusFunc(t.focus)
	tv.SetBlurFunc(t.blur)
	return t
}

// GetCurrentEntry returns the currently selected directory context
func (t *Tree) GetCurrentEntry() files.EntryWithDirPath {
	node := t.tv.GetCurrentNode()
	if node == nil {
		return nil
	}
	ref := node.GetReference()
	if ref == nil {
		return nil
	}
	dirContext, ok := ref.(*files.DirContext)
	if !ok || dirContext == nil {
		return nil
	}
	return dirContext
}
