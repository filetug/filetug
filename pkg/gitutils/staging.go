package gitutils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

func getWorktreeAndRelPath(path string) (*git.Worktree, string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	repoRoot, err := findRepoRoot(absPath)
	if err != nil {
		return nil, "", "", err
	}

	// On macOS /var is often a symlink to /private/var
	if realRepoRoot, err := filepath.EvalSymlinks(repoRoot); err == nil {
		repoRoot = realRepoRoot
	}
	if realAbsPath, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = realAbsPath
	}

	repo, err := git.PlainOpen(repoRoot)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to open git repo: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get worktree: %w", err)
	}

	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// go-git uses forward slashes
	relPath = filepath.ToSlash(relPath)

	return worktree, relPath, repoRoot, nil
}

// CanBeStaged checks if a file can be staged for a git commit.
// It returns true if the file is in a git repository and has changes (modified or untracked).
func CanBeStaged(path string) (bool, error) {
	worktree, relPath, _, err := getWorktreeAndRelPath(path)
	if err != nil {
		if err.Error() == "not in a git repository" {
			return false, nil // Not in a git repo
		}
		return false, err
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get git status: %w", err)
	}

	if status.IsClean() {
		return false, nil
	}

	fileStatus := status.File(relPath)
	if _, ok := status[relPath]; !ok {
		return false, nil
	}
	// Check if there are any changes in worktree or staging
	if fileStatus.Worktree != git.Unmodified || fileStatus.Staging != git.Unmodified {
		return true, nil
	}

	return false, nil
}

// StageFile stages a file for git commit.
func StageFile(path string) error {
	worktree, relPath, _, err := getWorktreeAndRelPath(path)
	if err != nil {
		return err
	}

	_, err = worktree.Add(relPath)
	if err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	return nil
}

// UnstageFile unstages a file from git commit.
func UnstageFile(path string) error {
	worktree, relPath, _, err := getWorktreeAndRelPath(path)
	if err != nil {
		return err
	}

	err = worktree.Reset(&git.ResetOptions{
		Files: []string{relPath},
	})
	if err != nil {
		return fmt.Errorf("failed to unstage file: %w", err)
	}

	return nil
}

// StageDir stages a directory for git commit.
func StageDir(path string, recursive bool) error {
	worktree, relPath, _, err := getWorktreeAndRelPath(path)
	if err != nil {
		return err
	}

	if recursive {
		_, err = worktree.Add(relPath)
		if err != nil {
			return fmt.Errorf("failed to stage directory recursively: %w", err)
		}
		return nil
	}

	// Non-recursive: we need to find files in the directory and add them
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			joinedPath := filepath.Join(relPath, entry.Name())
			fileRelPath := filepath.ToSlash(joinedPath)
			_, err = worktree.Add(fileRelPath)
			if err != nil {
				return fmt.Errorf("failed to stage file %s: %w", fileRelPath, err)
			}
		}
	}

	return nil
}

func findRepoRoot(path string) (string, error) {
	curr := path
	info, err := os.Stat(curr)
	if err == nil && !info.IsDir() {
		curr = filepath.Dir(curr)
	}

	for {
		gitPath := filepath.Join(curr, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", fmt.Errorf("not in a git repository")
		}
		curr = parent
	}
}
