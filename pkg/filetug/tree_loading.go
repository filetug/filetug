package filetug

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var spinner = []rune("▏▎▍▌▋▊▉█")

type loadingUpdater struct {
	node *tview.TreeNode
	text string
}

func (u loadingUpdater) Update() {
	u.node.SetText(u.text)
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

func (t *Tree) doLoadingAnimation(loading *tview.TreeNode) {
	nav := t.nav
	for {
		if children := t.rootNode.GetChildren(); len(children) == 0 || (len(children) == 1 && children[0] != loading) {
			nav.app.QueueUpdateDraw(func() {}) // For tests to signal completion
			return
		}
		q, r := t.loadingProgress/len(spinner), t.loadingProgress%len(spinner)
		progressBar := strings.Repeat("█", q) + string(spinner[r])
		updater := loadingUpdater{node: loading, text: " Loading... " + progressBar}
		nav.app.QueueUpdateDraw(updater.Update)
		t.loadingProgress += 1
		time.Sleep(50 * time.Millisecond)
	}
}
