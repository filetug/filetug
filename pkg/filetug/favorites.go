package filetug

import "github.com/rivo/tview"

type favorites struct {
	*tview.TreeView
}

func newFavorites() *favorites {
	f := &favorites{
		TreeView: tview.NewTreeView(),
	}

	favoritesNode := tview.NewTreeNode("Favorites").SetSelectable(false)
	f.SetRoot(favoritesNode)

	addFavNode := func(text, dir string) {
		favoritesNode.AddChild(tview.NewTreeNode(text).SetReference(dir))
	}

	addFavNode(" ~ [yellow]Alt+H[-][gray]ome[-]", "~")
	addFavNode(" / [yellow]Alt+R[-][gray]oot[-]", "/")
	return f
}
