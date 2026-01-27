package gitutils

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type FileGitStatus struct {
	Insertions int
	Deletions  int
}

func (s *FileGitStatus) String() string {
	var sb strings.Builder
	if s.Insertions > 0 {
		_, _ = fmt.Fprintf(&sb, "[green]+%d[-]", s.Insertions)
	}
	if s.Deletions > 0 {
		_, _ = fmt.Fprintf(&sb, "[red]-%d[-]", s.Deletions)
	}
	if sb.Len() == 0 {
		return "[lightgray]±0[-]"
	}
	return sb.String()
}

type DirGitChangesStats struct {
	FilesChanged int
	FileGitStatus
}

type RepoStatus struct {
	Branch string
	Err    error
	DirGitChangesStats
}

func (s *RepoStatus) String() string {
	const separator = "[gray]┆[-]"
	if s == nil {
		return ""
	}
	var noChanges DirGitChangesStats
	statusText := s.FileGitStatus.String()
	if s.DirGitChangesStats == noChanges {
		return separator + fmt.Sprintf("[darkgray]%s[-]%s", s.Branch, statusText)
	}
	return separator + fmt.Sprintf("[darkgray]%s[-]%s[darkgray]ƒ%d[-]%s", s.Branch, separator, s.FilesChanged, statusText)
}

// GetFileStatus returns a brief git status for a single file.
// It uses a context to allow cancellation and a semaphore to limit concurrency.
func GetFileStatus(ctx context.Context, repo *git.Repository, filePath string) *RepoStatus {
	if repo == nil {
		return nil
	}

	wt, err := repoWorktree(repo)
	if err != nil {
		return nil
	}
	repoRoot := wt.Filesystem.Root()
	lock := getRepoLock(repoRoot)
	lock.Lock()
	defer lock.Unlock()

	select {
	case <-ctx.Done():
		return nil
	case gitStatusSemaphore <- struct{}{}:
		defer func() { <-gitStatusSemaphore }()
	}

	res := &RepoStatus{}

	head, err := repoHead(repo)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) || err.Error() == "reference not found" {
			res.Branch = "master"
		} else {
			res.Branch = "unknown"
		}
	} else if head == nil || head.Hash().IsZero() {
		res.Branch = "unknown"
	} else {
		if head.Name().IsBranch() {
			res.Branch = head.Name().Short()
		} else {
			res.Branch = shortHash(head.Hash().String())
		}
	}

	select {
	case <-ctx.Done():
		return res
	default:
	}

	var headHash plumbing.Hash
	if head != nil {
		headHash = head.Hash()
	}
	headCommit, _ := repo.CommitObject(headHash)

	worktree, err := repoWorktree(repo)
	if err != nil {
		return res
	}

	status, err := worktreeStatus(worktree)
	if err != nil {
		return res
	}

	if status.IsClean() {
		return res
	}

	relPath, err := filepathRel(repoRoot, filePath)
	if err != nil {
		return res
	}
	relPath = filepath.ToSlash(relPath)

	fileStatus, ok := status[relPath]
	if !ok {
		return res
	}
	if fileStatus.Worktree == git.Unmodified && fileStatus.Staging == git.Unmodified {
		return res
	}

	res.FilesChanged = 1

	if headCommit != nil {
		headTree, err := headCommit.Tree()
		if err == nil {
			if isCtxDone(ctx) {
				return res
			}

			if fileStatus.Worktree == git.Untracked {
				if f, err := worktree.Filesystem.Open(relPath); err == nil {
					content, err := readLimitedContentFn(f)
					_ = f.Close()
					if err != nil {
						return res
					}
					res.Insertions += countLines(content)
				}
			}

			if fileStatus.Worktree == git.Deleted || fileStatus.Staging == git.Deleted {
				if f, err := headTree.File(relPath); err == nil {
					if content, err := readHeadFileContents(f); err == nil {
						res.Deletions += countLines(content)
					}
				}
			}

			if fileStatus.Worktree == git.Untracked || fileStatus.Worktree == git.Deleted || fileStatus.Staging == git.Deleted {
				return res
			}

			var headContent string
			headFile, err := headTree.File(relPath)
			if err != nil {
				if f, err := worktree.Filesystem.Open(relPath); err == nil {
					content, err := readLimitedContent(f)
					_ = f.Close()
					if err == nil {
						res.Insertions += countLines(content)
					}
				}
				return res
			}
			headContent, err = readHeadFileContents(headFile)
			if err != nil {
				return res
			}

			f, err := worktree.Filesystem.Open(relPath)
			if err != nil {
				return res
			}
			worktreeContent, err := readLimitedContentFn(f)
			_ = f.Close()
			if err != nil {
				return res
			}

			insertions, deletions := diffLineStats(headContent, worktreeContent)
			res.Insertions += insertions
			res.Deletions += deletions
		}
	}

	return res
}
