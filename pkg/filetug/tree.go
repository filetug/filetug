package filetug

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

const dirEmoji = "📁"

type Tree struct {
	*sneatv.Boxed
	tv              *tview.TreeView
	nav             *Navigator
	rootNode        *tview.TreeNode
	searchPattern   string
	loadingProgress int
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
		if t.nav != nil && t.nav.queueUpdateDraw != nil {
			t.nav.queueUpdateDraw(func() {
				loading.SetText(" Loading... " + progressBar)
			})
		} else {
			loading.SetText(" Loading... " + progressBar)
		}
		t.loadingProgress += 1
		t.doLoadingAnimation(loading)
	}
}

func (t *Tree) Draw(screen tcell.Screen) {
	root := t.tv.GetRoot()
	text := root.GetText()
	if strings.HasSuffix(text, " ") {
		_, _, width, _ := t.tv.GetInnerRect()
		if width > len(text) {
			text += strings.Repeat(" ", width-len(text))
			root.SetText(text)
		}
	}
	t.Boxed.Draw(screen)
}

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
	t.nav.queueUpdateDraw = nav.queueUpdateDraw
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

	errText := fmt.Sprintf("%s: %v", name, err)
	text := dirEmoji + errText
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
	currentNode := t.tv.GetCurrentNode()
	if currentNode == nil {
		currentNode = t.tv.GetRoot()
		t.tv.SetCurrentNode(currentNode)
	}
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(sneatv.CurrentTheme.FocusedSelectedTextStyle)
	}
	t.tv.SetGraphicsColor(tcell.ColorWhite)
}

func (t *Tree) blur() {
	t.nav.left.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	t.tv.SetGraphicsColor(sneatv.CurrentTheme.BlurredGraphicsColor)
	currentNode := t.tv.GetCurrentNode()
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(sneatv.CurrentTheme.BlurredSelectedTextStyle)
	}
}

func (t *Tree) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRight:
		t.nav.setAppFocus(t.nav.files)
		return nil
	case tcell.KeyLeft:
		currentNode := t.tv.GetCurrentNode()
		refValue := currentNode.GetReference()
		switch ref := refValue.(type) {
		case string:
			parentDir, _ := path.Split(ref)
			t.nav.goDir(parentDir)
			return nil
		}
		return event
	case tcell.KeyEnter:
		currentNode := t.tv.GetCurrentNode()
		switch ref := currentNode.GetReference().(type) {
		case string:
			if ref != "/" {
				ref = strings.TrimSuffix(ref, "/")
			}
			if currentNode == t.tv.GetRoot() {
				expandedRef := fsutils.ExpandHome(ref)
				ref, _ = path.Split(expandedRef)
			}
			t.nav.goDir(ref)
			return nil
		}
		return event
	case tcell.KeyUp:
		if t.tv.GetCurrentNode() == t.tv.GetRoot() {
			t.nav.breadcrumbs.TakeFocus(t.tv)
			t.nav.setAppFocus(t.nav.breadcrumbs)
			return nil
		}
		return event
	case tcell.KeyBackspace:
		t.SetSearch(t.searchPattern[:len(t.searchPattern)-1])
		return nil
	case tcell.KeyEscape:
		t.SetSearch("")
		return nil
	case tcell.KeyRune:
		s := string(event.Rune())
		if t.searchPattern == "" && s == " " {
			return event
		}
		lower := strings.ToLower(s)
		t.SetSearch(t.searchPattern + lower)
		return nil
	default:
		return event
	}
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
	highlightTreeNodes(root, searchCtx)
	if searchCtx.firstPrefixed != nil {
		t.tv.SetCurrentNode(searchCtx.firstPrefixed)
	} else if searchCtx.firstContains != nil {
		t.tv.SetCurrentNode(searchCtx.firstContains)
	} else if len(t.searchPattern) > 0 {
		t.SetSearch(t.searchPattern[:len(t.searchPattern)-1])
	}
}

type searchContext struct {
	pattern       string
	found         []string
	firstContains *tview.TreeNode
	firstPrefixed *tview.TreeNode
}

func highlightTreeNodes(n *tview.TreeNode, searchCtx *searchContext) {
	r := n.GetReference()
	if s, ok := r.(string); ok {
		_, name := path.Split(s)
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
	for _, child := range n.GetChildren() {
		highlightTreeNodes(child, searchCtx)
	}
}

var userHomeDir, _ = os.UserHomeDir()

func (t *Tree) setCurrentDir(dir string) {
	t.SetSearch("")
	t.rootNode.ClearChildren()

	root := t.nav.store.RootURL()
	if root.Path == "" {
		root.Path = "/"
	}

	var panelTitle, text string
	if dir == root.Path {
		if dir == "/" {
			text = "/"
		} else {
			text = strings.TrimSuffix(root.Path, "/")
		}
	} else {
		text = ".."
		trimmedDir := strings.TrimSuffix(dir, "/")
		_, panelTitle = path.Split(trimmedDir)
		if root.Scheme == "file" && trimmedDir == userHomeDir {
			panelTitle = "~"
		}
	}
	t.SetTitle(panelTitle)

	t.rootNode.SetText(text)
	t.rootNode.SetReference(dir)
	t.rootNode.SetColor(tcell.ColorWhite)
}

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

func (t *Tree) GetCurrentEntry() *files.EntryWithDirPath {
	node := t.tv.GetCurrentNode()
	if node == nil {
		return nil
	}
	ref := node.GetReference()
	if ref == nil {
		return nil
	}
	p := ref.(string)
	baseName := path.Base(p)
	return &files.EntryWithDirPath{
		Dir:      path.Dir(p),
		DirEntry: &treeDirEntry{name: baseName, isDir: true},
	}
}

func (t *Tree) setDirContext(ctx context.Context, node *tview.TreeNode, dirContext *DirContext) {
	node.ClearChildren()

	var repo *git.Repository
	if t.nav.store.RootURL().Scheme == "file" {
		repoRoot := gitutils.GetRepositoryRoot(dirContext.Path)
		if repoRoot != "" {
			repo, _ = git.PlainOpen(repoRoot)
		}
	}

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
			n := tview.NewTreeNode(prefix)
			n.SetReference(childPath)
			node.AddChild(n)

			fullPath := fsutils.ExpandHome(childPath)
			go t.nav.updateGitStatus(ctx, repo, fullPath, n, prefix+" ")
		}
	}
}
