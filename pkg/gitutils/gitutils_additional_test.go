package gitutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestGetDirStatus_WorktreeError(t *testing.T) {
	//t.Parallel()
	origRepoWorktree := repoWorktree

	defer func() {
		repoWorktree = origRepoWorktree
	}()

	repoWorktree = func(_ *git.Repository) (*git.Worktree, error) {
		return nil, fmt.Errorf("worktree error")
	}

	status := GetDirStatus(context.Background(), &git.Repository{}, "/tmp")
	assert.Nil(t, status)
}

func TestGetDirStatus_WorktreeStatusErrorSeam(t *testing.T) {
	//t.Parallel()
	origWorktreeStatus := worktreeStatus

	defer func() {
		worktreeStatus = origWorktreeStatus
	}()

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	worktreeStatus = func(_ *git.Worktree) (git.Status, error) {
		return nil, fmt.Errorf("status error")
	}

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
	assert.Equal(t, 0, status.FilesChanged)
}

func TestGetDirStatus_SecondWorktreeError(t *testing.T) {
	//t.Parallel()
	origRepoWorktree := repoWorktree

	defer func() {
		repoWorktree = origRepoWorktree
	}()

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	firstWorktree, err := repo.Worktree()
	assert.NoError(t, err)

	callCount := 0
	repoWorktree = func(_ *git.Repository) (*git.Worktree, error) {
		callCount++
		if callCount == 1 {
			return firstWorktree, nil
		}
		return nil, fmt.Errorf("worktree error")
	}

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
}

func TestGetDirStatus_FilepathRelError(t *testing.T) {
	//t.Parallel()
	origFilepathRel := filepathRel
	origExecCommand := execCommand

	defer func() {
		filepathRel = origFilepathRel
		execCommand = origExecCommand
	}()

	execCommand = fakeExecCommand("", 1)
	filepathRel = func(_, _ string) (string, error) {
		return "", fmt.Errorf("rel error")
	}

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("content\n"), 0o644)
	assert.NoError(t, err)

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
	assert.Equal(t, 1, status.FilesChanged)
}

func TestGetDirStatus_ContextDoneDuringDiff(t *testing.T) {
	//t.Parallel()
	origIsCtxDone := isCtxDone
	origExecCommand := execCommand

	defer func() {
		isCtxDone = origIsCtxDone
		execCommand = origExecCommand
	}()

	execCommand = fakeExecCommand("", 1)
	isCtxDone = func(_ context.Context) bool {
		return true
	}

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	worktree, err := repo.Worktree()
	assert.NoError(t, err)

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("line1\n"), 0o644)
	assert.NoError(t, err)
	_, err = worktree.Add("file.txt")
	assert.NoError(t, err)
	_, err = worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	assert.NoError(t, err)

	err = os.WriteFile(filePath, []byte("line1\nline2\n"), 0o644)
	assert.NoError(t, err)

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
	assert.Equal(t, 1, status.FilesChanged)
}

func TestGetDirStatus_IgnoresPathsDuringDiff(t *testing.T) {
	//t.Parallel()
	origExecCommand := execCommand

	defer func() {
		execCommand = origExecCommand
	}()

	execCommand = fakeExecCommand("", 1)

	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	worktree, err := repo.Worktree()
	assert.NoError(t, err)

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("line1\n"), 0o644)
	assert.NoError(t, err)
	_, err = worktree.Add("file.txt")
	assert.NoError(t, err)
	_, err = worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	assert.NoError(t, err)

	err = os.WriteFile(filePath, []byte("line1\nline2\n"), 0o644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, ".DS_Store"), []byte("ignored\n"), 0o644)
	assert.NoError(t, err)

	status := GetDirStatus(context.Background(), repo, tempDir)
	assert.NotNil(t, status)
	assert.Equal(t, 1, status.FilesChanged)
}
