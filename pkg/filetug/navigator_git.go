package filetug

import (
	"context"
	"strings"

	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

const gitStatusSeparator = "[gray]┆[-]"

func (nav *Navigator) updateGitStatus(ctx context.Context, repo *git.Repository, fullPath string, node *tview.TreeNode, prefix string) {
	cleanPrefix := stripGitStatusPrefix(prefix)
	if node == nil {
		return
	}
	status := nav.getGitStatus(ctx, repo, fullPath, true)
	if status == nil {
		return
	}
	statusText := nav.gitStatusText(status, fullPath, true)
	if statusText == "" {
		return
	}
	nav.app.QueueUpdateDraw(func() {
		node.SetText(cleanPrefix + statusText)
	})
}

func stripGitStatusPrefix(text string) string {
	separatorIndex := strings.Index(text, gitStatusSeparator)
	if separatorIndex == -1 {
		return text
	}
	return text[:separatorIndex]
}

func (nav *Navigator) getGitStatus(ctx context.Context, repo *git.Repository, fullPath string, isDir bool) *gitutils.RepoStatus {
	nav.gitStatusCacheMu.RLock()
	cachedStatus, ok := nav.gitStatusCache[fullPath]
	nav.gitStatusCacheMu.RUnlock()
	if ok {
		return cachedStatus
	}

	if repo == nil {
		repoRoot := gitutils.GetRepositoryRoot(fullPath)
		if repoRoot == "" {
			return nil
		}

		var err error
		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return nil
		}
	}

	var status *gitutils.RepoStatus
	if isDir {
		status = getDirStatus(ctx, repo, fullPath)
	} else {
		status = getFileStatus(ctx, repo, fullPath)
	}
	if status == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	nav.gitStatusCacheMu.Lock()
	nav.gitStatusCache[fullPath] = status
	nav.gitStatusCacheMu.Unlock()
	return status
}

func (nav *Navigator) gitStatusText(status *gitutils.RepoStatus, fullPath string, isDir bool) string {
	if status == nil {
		return ""
	}

	hasChanges := status.FilesChanged > 0 || status.Insertions > 0 || status.Deletions > 0
	isRepoRoot := false
	if isDir {
		repoRoot := gitutils.GetRepositoryRoot(fullPath)
		isRepoRoot = repoRoot != "" && (fullPath == repoRoot || fullPath == repoRoot+"/")
	}
	if hasChanges || isRepoRoot {
		return status.String()
	}
	return ""
}
