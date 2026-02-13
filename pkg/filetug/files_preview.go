package filetug

import (
	"context"
	"path"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
)

// updatePreviewForEntry updates the preview panel to show the selected entry.
// For directories, it shows a directory summary. For files, it shows file preview.
func (f *filesPanel) updatePreviewForEntry(entry files.EntryWithDirPath) {
	nav := f.nav
	if nav == nil {
		return
	}
	isDir := entry.IsDir()
	if !isDir && f.rows != nil {
		isDir = f.rows.isSymlinkToDir(entry)
	}
	if isDir {
		f.showDirSummary(entry)
		return
	}

	if nav.right != nil && nav.previewer != nil {
		nav.right.SetContent(nav.previewer)
	}
	fullName := entry.FullName()
	f.rememberCurrent(fullName)
	if nav.previewer == nil {
		return
	}
	nav.previewer.PreviewEntry(entry)
}

// showDirSummary displays a summary of the selected directory in the preview panel.
// It reads the directory contents and displays statistics about the files within.
func (f *filesPanel) showDirSummary(entry files.EntryWithDirPath) {
	nav := f.nav
	if nav == nil {
		return
	}
	if nav.right != nil && nav.previewer != nil {
		nav.right.SetContent(nav.previewer)
		nav.previewer.PreviewEntry(entry)
	}

	dirPath := entry.DirPath()
	if entry.IsDir() {
		dirPath = entry.FullName()
	} else if f.rows != nil && f.rows.isSymlinkToDir(entry) {
		dirPath = entry.FullName()
	}

	if nav.store == nil {
		dirContext := files.NewDirContext(nil, dirPath, nil)
		if nav.previewer != nil && nav.previewer.dirPreviewer != nil {
			nav.previewer.dirPreviewer.SetDirEntries(dirContext)
		}
		return
	}
	ctx := context.Background()
	entries, err := nav.store.ReadDir(ctx, dirPath)
	if err != nil {
		dirContext := files.NewDirContext(nav.store, dirPath, nil)
		if nav.previewer != nil && nav.previewer.dirPreviewer != nil {
			nav.previewer.dirPreviewer.SetDirEntries(dirContext)
		}
		return
	}
	sortedEntries := sortDirChildren(entries)
	dirContext := files.NewDirContext(nav.store, dirPath, sortedEntries)
	if nav.previewer != nil && nav.previewer.dirPreviewer != nil {
		nav.previewer.dirPreviewer.SetDirEntries(dirContext)
	}
}

// rememberCurrent saves the current filename to state so it can be restored
// when returning to this directory.
func (f *filesPanel) rememberCurrent(fullName string) {
	_, f.currentFileName = path.Split(fullName)
	ftstate.SaveCurrentFileName(f.currentFileName)
}
