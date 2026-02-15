package filetug

import (
	"os"
	"path"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/gdamore/tcell/v2"
	"github.com/strongo/strongo-tui/pkg/themes"
)

var userHomeDir, _ = os.UserHomeDir()

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

func (t *Tree) focus() {
	t.nav.left.SetBorderColor(themes.CurrentTheme.FocusedBorderColor())
	t.nav.activeCol = 0
	t.nav.right.SetContent(t.nav.previewer)
	t.nav.previewer.Blur()
	t.nav.right.Blur()
	t.nav.files.blur()
	currentNode := t.tv.GetCurrentNode()
	if currentNode == nil {
		currentNode = t.tv.GetRoot()
		t.tv.SetCurrentNode(currentNode)
	}
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(themes.CurrentTheme.FocusedSelectedTextStyle())
	}
	t.tv.SetGraphicsColor(tcell.ColorWhite)
}

func (t *Tree) blur() {
	t.nav.left.SetBorderColor(themes.CurrentTheme.BlurredBorderColor())
	t.tv.SetGraphicsColor(themes.CurrentTheme.BlurredGraphicsColor())
	currentNode := t.tv.GetCurrentNode()
	if currentNode != nil {
		currentNode.SetSelectedTextStyle(themes.CurrentTheme.BlurredSelectedTextStyle())
	}
}

func (t *Tree) setCurrentDir(dirContext *files.DirContext) {
	if dirContext == nil {
		return
	}
	t.SetSearch("")
	t.rootNode.ClearChildren()

	root := t.nav.store.RootURL()
	if root.Path == "" {
		root.Path = "/"
	}

	var panelTitle, text string
	if dirContext.Path() == root.Path {
		if dirContext.Path() == "/" {
			text = "/"
		} else {
			text = strings.TrimSuffix(root.Path, "/")
		}
	} else {
		text = ".."
		trimmedDir := strings.TrimSuffix(dirContext.Path(), "/")
		_, panelTitle = path.Split(trimmedDir)
		if root.Scheme == "file" && trimmedDir == userHomeDir {
			panelTitle = "~"
		}
	}
	t.SetTitle(panelTitle)

	t.rootNode.SetText(text)
	t.rootNode.SetReference(dirContext)
	t.rootNode.SetColor(tcell.ColorWhite)
}
