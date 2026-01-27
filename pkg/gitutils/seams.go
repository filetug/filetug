package gitutils

import (
	"context"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	filepathAbs          = filepath.Abs
	filepathRel          = filepath.Rel
	filepathEvalSymlinks = filepath.EvalSymlinks

	gitPlainOpen = git.PlainOpen

	repoHead = func(repo *git.Repository) (*plumbing.Reference, error) {
		return repo.Head()
	}
	repoWorktree = func(repo *git.Repository) (*git.Worktree, error) {
		return repo.Worktree()
	}
	worktreeStatus = func(wt *git.Worktree) (git.Status, error) {
		return wt.Status()
	}
	worktreeAdd = func(wt *git.Worktree, path string) (plumbing.Hash, error) {
		return wt.Add(path)
	}
	readLimitedContentFn = readLimitedContent
	readHeadFileContents = func(f *object.File) (string, error) {
		return f.Contents()
	}

	isCtxDone = func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	}
)
