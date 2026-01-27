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

	var headHash plumbing.Hash
	head, err := repoHead(repo)
	if err != nil {
		res.Err = err
		if errors.Is(err, plumbing.ErrReferenceNotFound) || err.Error() == "reference not found" {
			res.Branch = "master"
		} else {
			res.Branch = "unknown"
		}
	}
	if head != nil {
		headHash = head.Hash()
	}
	if res.Branch == "" {
		if head == nil {
			res.Branch = "unknown"
		} else if head.Name().IsBranch() {
			res.Branch = head.Name().Short()
		} else if headHash.IsZero() {
			res.Branch = "unknown"
		} else {
			res.Branch = "{HEAD detached at " + shortHash(headHash.String()) + "}"
		}
	}

	select {
	case <-ctx.Done():
		return res
	default:
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

	matcher := LoadGlobalIgnoreMatcher(repoRoot)

	relPath, err := filepathRel(repoRoot, dir)
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
		fileNameSlash := filepath.ToSlash(fileName)
		if IsIgnoredPath(fileNameSlash, matcher) {
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
				if isCtxDone(ctx) {
					return res
				}

				if relPath != "" && !strings.HasPrefix(fileName, relPath) {
					continue
				}
				fileNameSlash := filepath.ToSlash(fileName)
				if IsIgnoredPath(fileNameSlash, matcher) {
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
