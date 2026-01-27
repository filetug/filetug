package gitutils

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	// gitStatusSemaphore limits concurrent git status calls to avoid system hang
	gitStatusSemaphore = make(chan struct{}, 2)

	repoLocksMu sync.Mutex
	repoLocks   = make(map[string]*sync.Mutex)
)

func getRepoLock(repoPath string) *sync.Mutex {
	repoLocksMu.Lock()
	defer repoLocksMu.Unlock()
	if lock, ok := repoLocks[repoPath]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	repoLocks[repoPath] = lock
	return lock
}

// GetDirStatus returns a brief git status for the given directory.
// It uses a context to allow cancellation and a semaphore to limit concurrency.
func GetDirStatus(ctx context.Context, repo *git.Repository, dir string) *RepoStatus {
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
			// This covers some other error during repo.Head()
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

	relPath, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		relPath = ""
	}
	if relPath == "." {
		relPath = ""
	}
	separator := string(filepath.Separator)
	if relPath != "" && !strings.HasSuffix(relPath, separator) {
		relPath += separator
	}

	res.FilesChanged = 0
	for fileName, s := range status {
		if relPath != "" && !strings.HasPrefix(fileName, relPath) {
			continue
		}
		if s.Worktree != git.Unmodified || s.Staging != git.Unmodified {
			res.FilesChanged++
		}
	}

	if res.FilesChanged == 0 {
		return res
	}

	// To get insertions/deletions, we need to diff
	if headCommit != nil {
		headTree, err := headCommit.Tree()
		if err == nil {
			for fileName, fileStatus := range status {
				select {
				case <-ctx.Done():
					return res
				default:
				}

				if relPath != "" && !strings.HasPrefix(fileName, relPath) {
					continue
				}

				if fileStatus.Worktree == git.Unmodified && fileStatus.Staging == git.Unmodified {
					continue
				}

				// If file is untracked, we can count its lines as insertions
				if fileStatus.Worktree == git.Untracked {
					if f, err := worktree.Filesystem.Open(fileName); err == nil {
						// Limit reading to avoid performance issues with large files
						const maxRead = 1 * 1024 * 1024 // 1MB
						b := make([]byte, maxRead)
						n, _ := f.Read(b)
						content := string(b[:n])
						res.Insertions += strings.Count(content, "\n")
						_ = f.Close()
					}
					continue
				}

				// If file is deleted, we can count its lines in head as deletions
				if fileStatus.Worktree == git.Deleted || fileStatus.Staging == git.Deleted {
					if f, err := headTree.File(fileName); err == nil {
						if content, err := f.Contents(); err == nil {
							res.Deletions += strings.Count(content, "\n")
						}
					}
					continue
				}
			}
		}
	}

	return res
}
