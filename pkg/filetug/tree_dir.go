package filetug

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

const dirEmoji = "📁"

func dirNodeText(name string) string {
	return emojiForDir(name) + " " + name
}

func emojiForDir(name string) string {
	switch strings.ToLower(name) {
	case "library":
		return "📚"
	case "users":
		return "👥"
	case "applications":
		return "🈸"
	case "music":
		return "🎹"
	case "movies":
		return "📺"
	case "pictures":
		return "🖼️"
	case "desktop":
		return "🖥️"
	case "datatug":
		return "🛥️"
	case "documents":
		return "🗃"
	case "public":
		return "📢"
	case "temp":
		return "⏳"
	case "system":
		return "🧠"
	case "bin", "sbin":
		return "🚀"
	case "private":
		return "🔒"
	default:
		return dirEmoji
	}
}

func (t *Tree) changed(node *tview.TreeNode) {
	ref := node.GetReference()

	if dirContext, ok := ref.(*files.DirContext); ok {
		var ctx context.Context
		ctx, t.nav.cancel = context.WithCancel(context.Background())
		t.nav.showDir(ctx, node, dirContext, false)
		ftstate.SaveSelectedTreeDir(dirContext.Path())
	}
}

func (t *Tree) setError(node *tview.TreeNode, err error) {
	//panic(err)
	node.ClearChildren()
	node.SetColor(tcell.ColorOrangeRed)
	nodePath := getNodePath(node)
	_, name := path.Split(nodePath)

	errText := fmt.Sprintf("%s: %v", name, err)
	text := dirEmoji + errText
	node.SetText(text)
	//node.AddChild(tview.NewTreeNode(err.Error()))
}

func getNodePath(node *tview.TreeNode) string {
	if node == nil {
		return ""
	}
	ref := node.GetReference()
	dirContext, ok := ref.(*files.DirContext)
	if !ok || dirContext == nil {
		return ""
	}
	return dirContext.Path()
}

func (t *Tree) setDirContext(ctx context.Context, node *tview.TreeNode, dirContext *files.DirContext) {
	node.ClearChildren()

	var repo *git.Repository
	if t.nav.store.RootURL().Scheme == "file" {
		repoRoot := gitutils.GetRepositoryRoot(dirContext.Path())
		if repoRoot != "" {
			repo, _ = git.PlainOpen(repoRoot)
		}
	}

	children := dirContext.Children()
	for _, child := range children {
		name := child.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if child.IsDir() {
			childPath := path.Join(dirContext.Path(), name)
			prefix := dirNodeText(name)
			n := tview.NewTreeNode(prefix).SetIndent(1)
			childContext := files.NewDirContext(dirContext.Store(), childPath, nil)
			n.SetReference(childContext)
			node.AddChild(n)

			fullPath := fsutils.ExpandHome(childPath)
			go t.nav.updateGitStatus(ctx, repo, fullPath, n, prefix+" ")
		}
	}
}
