package filetug

import (
	"context"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
)

// updateGitStatuses asynchronously updates git status indicators for all entries
// in the files panel. It uses a worker pool to efficiently process git status checks.
func (f *filesPanel) updateGitStatuses(ctx context.Context, dirContext *files.DirContext) {
	if f.nav == nil || f.rows == nil || dirContext == nil {
		return
	}
	if f.nav.store == nil || f.nav.store.RootURL().Scheme != "file" {
		return
	}
	repoRoot := gitutils.GetRepositoryRoot(dirContext.Path())
	if repoRoot == "" {
		return
	}
	repo, err := git.PlainOpen(repoRoot)
	if err != nil {
		return
	}

	rows := f.rows
	table := f.table
	queueUpdateDraw := f.nav.app.QueueUpdateDraw
	pool := gitutils.GetGlobalPool()
	
	for _, entry := range rows.AllEntries {
		entry := entry
		fullPath := entry.FullName()
		isDir := entry.IsDir()
		if !isDir {
			isDir = rows.isSymlinkToDir(entry)
		}

		req := gitutils.GitStatusRequest{
			Repo:  repo,
			Path:  fullPath,
			IsDir: isDir,
			Callback: func(status *gitutils.RepoStatus) {
				if status == nil {
					return
				}
				statusText := f.nav.gitStatusText(status, fullPath, isDir)
				if statusText == "" {
					return
				}
				updated := rows.SetGitStatusText(fullPath, statusText)
				if !updated {
					return
				}
				queueUpdateDraw(func() {
					if f.rows != rows {
						return
					}
					table.SetContent(rows)
				})
			},
		}
		pool.Submit(req)
	}
}
