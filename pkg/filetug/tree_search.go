package filetug

import (
	"fmt"
	"path"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/rivo/tview"
)

type searchContext struct {
	pattern       string
	found         []string
	firstContains *tview.TreeNode
	firstPrefixed *tview.TreeNode
}

func (t *Tree) SetSearch(pattern string) {
	t.searchPattern = pattern
	if pattern == "" {
		t.searchPattern = ""
		t.SetTitle("")
	} else {
		title := fmt.Sprintf("Find: %s", t.searchPattern)
		t.SetTitle(title)
	}
	searchCtx := &searchContext{
		pattern: t.searchPattern,
	}
	root := t.tv.GetRoot()
	highlightTreeNodes(root, searchCtx, true)
	if searchCtx.firstPrefixed != nil {
		t.tv.SetCurrentNode(searchCtx.firstPrefixed)
	} else if searchCtx.firstContains != nil {
		t.tv.SetCurrentNode(searchCtx.firstContains)
	} else if len(t.searchPattern) > 0 {
		t.SetSearch(t.searchPattern[:len(t.searchPattern)-1])
	}
}

func highlightTreeNodes(n *tview.TreeNode, searchCtx *searchContext, isRoot bool) {
	if !isRoot {
		r := n.GetReference()
		if dirContext, ok := r.(*files.DirContext); ok {
			_, name := path.Split(dirContext.Path())
			lowerName := strings.ToLower(name)
			if strings.Contains(lowerName, searchCtx.pattern) {
				i := strings.Index(lowerName, searchCtx.pattern)
				ss := name[i : i+len(searchCtx.pattern)]
				formatted := fmt.Sprintf("[black:lightgreen]%s[-:-]", ss)
				text := strings.ReplaceAll(name, ss, formatted)
				n.SetText(text)
				searchCtx.found = append(searchCtx.found, text)
				if searchCtx.firstContains == nil {
					searchCtx.firstContains = n
				}
				if searchCtx.firstPrefixed == nil && strings.HasPrefix(lowerName, searchCtx.pattern) {
					searchCtx.firstPrefixed = n
				}
			} else {
				n.SetText(name)
			}
		}
	}
	for _, child := range n.GetChildren() {
		highlightTreeNodes(child, searchCtx, false)
	}
}
