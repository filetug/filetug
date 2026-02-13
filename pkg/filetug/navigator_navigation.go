package filetug

import (
	"context"
	"errors"
	"os"
	"sort"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

func (nav *Navigator) goRoot() {
	nav.goDirByPath("/")
}

func (nav *Navigator) goHome() {
	nav.goDirByPath("~")
}

func (nav *Navigator) goDirByPath(dirPath string) {
	dirContext := files.NewDirContext(nav.store, dirPath, nil)
	nav.goDir(dirContext)
}

func (nav *Navigator) goDir(dirContext *files.DirContext) {
	if dirContext == nil {
		return
	}
	nav.dirsTree.setCurrentDir(dirContext)
	ctx := context.Background()
	nav.showDir(ctx, nav.dirsTree.rootNode, dirContext, true)
	root := nav.store.RootURL()
	rootValue := root.String()
	nav.saveCurrentDir(rootValue, dirContext.Path())
}

// showDir updates all panels.
// The `isTreeRootChanged bool` argument is needed do distinguish root dir change from
// the case we simply select the root node in the tree.
func (nav *Navigator) showDir(ctx context.Context, node *tview.TreeNode, dirContext *files.DirContext, isTreeRootChanged bool) {
	if dirContext == nil {
		return
	}
	expandedDir := fsutils.ExpandHome(dirContext.Path())
	if expandedDir != dirContext.Path() {
		dirContext = files.NewDirContext(dirContext.Store(), expandedDir, dirContext.Children())
	}
	currentDirPath := nav.currentDirPath()
	if currentDirPath == expandedDir && !isTreeRootChanged {
		return // TODO: Investigate and document why this happens or fix
	}
	currentChildren := dirContext.Children()
	currentDirContext := files.NewDirContext(dirContext.Store(), expandedDir, currentChildren)
	nav.current.SetDir(currentDirContext)
	if node != nil {
		node.SetReference(dirContext)
	}
	if nav.store != nil && nav.store.RootURL().Scheme == "file" && node != nil {
		name := node.GetText()
		currentDir := nav.current.Dir()
		if currentDir != nil {
			repoRoot := gitutils.GetRepositoryRoot(currentDir.Path())
			var repo *git.Repository
			if repoRoot != "" {
				repo, _ = git.PlainOpen(repoRoot)
			}
			currentPath := currentDir.Path()
			go nav.updateGitStatus(ctx, repo, currentPath, node, name)
		}
	}

	nav.setBreadcrumbs()
	if nav.right != nil {
		nav.right.SetContent(nav.previewer)
	}

	dirPath := expandedDir
	// Start loading data in a goroutine
	go func() {
		dirContext, err := nav.getDirData(ctx, dirPath)
		if nav.app != nil {
			nav.app.QueueUpdateDraw(func() {
				if err != nil {
					nav.showNodeError(node, err)
					return
				}
				nav.onDataLoaded(ctx, node, dirContext, isTreeRootChanged)
			})
		}
	}()
}

func (nav *Navigator) onDataLoaded(ctx context.Context, node *tview.TreeNode, dirContext *files.DirContext, isTreeRootChanged bool) {
	if nav.previewer != nil {
		nav.previewer.PreviewEntry(dirContext)
	}

	//nav.filesPanel.Clear()
	if nav.files != nil {
		nav.files.table.SetSelectable(true, false)

		dirRecords := NewFileRows(dirContext)
		nav.files.SetRows(dirRecords, node != nil && node != nav.dirsTree.rootNode)
	}

	if isTreeRootChanged && node != nil && nav.dirsTree != nil {
		nav.dirsTree.setDirContext(ctx, node, dirContext)
	}
	if nav.files != nil {
		nav.files.updateGitStatuses(ctx, dirContext)
	}
}

func (nav *Navigator) getDirData(ctx context.Context, dirPath string) (dirContext *files.DirContext, err error) {
	if nav.store == nil {
		return nil, errors.New("store not set")
	}
	dirContext = files.NewDirContext(nav.store, dirPath, nil)
	var children []os.DirEntry
	children, err = nav.store.ReadDir(ctx, dirPath)
	if err != nil {
		return nil, err
	}
	// Tree is always sorted by name and files are usually as well
	// So let's sort once here and pass sorted to both Tree and filesPanel.
	children = sortDirChildren(children)
	//time.Sleep(time.Millisecond * 2000)
	dirContext.SetChildren(children)
	return
}

func sortDirChildren(children []os.DirEntry) []os.DirEntry {
	sort.Slice(children, func(i, j int) bool {
		// Directories first
		if children[i].IsDir() && !children[j].IsDir() {
			return true
		} else if !children[i].IsDir() && children[j].IsDir() {
			return false
		}
		// Then sort by name
		return children[i].Name() < children[j].Name()
	})
	return children
}
