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

	wt, err := repo.Worktree()
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

	head, err := repo.Head()
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
			hashStr := head.Hash().String()
			if len(hashStr) >= 7 {
				res.Branch = hashStr[:7]
			} else {
				res.Branch = hashStr
			}
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

	worktree, err := repo.Worktree()
	if err != nil {
		return res
	}

	status, err := worktree.Status()
	if err != nil {
		return res
	}

	if status.IsClean() {
		return res
	}

	relPath, err := filepath.Rel(repoRoot, filePath)
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
			select {
			case <-ctx.Done():
				return res
			default:
			}

			if fileStatus.Worktree == git.Untracked {
				if f, err := worktree.Filesystem.Open(relPath); err == nil {
					const maxRead = 1 * 1024 * 1024
					b := make([]byte, maxRead)
					n, _ := f.Read(b)
					content := string(b[:n])
					res.Insertions += strings.Count(content, "\n")
					_ = f.Close()
				}
			}

			if fileStatus.Worktree == git.Deleted || fileStatus.Staging == git.Deleted {
				if f, err := headTree.File(relPath); err == nil {
					if content, err := f.Contents(); err == nil {
						res.Deletions += strings.Count(content, "\n")
					}
				}
			}
		}
	}

	return res
}
