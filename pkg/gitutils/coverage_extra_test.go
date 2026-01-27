package gitutils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func stubRepoHead(t *testing.T, stub func(*git.Repository) (*plumbing.Reference, error)) {
	old := repoHead
	repoHead = stub
	t.Cleanup(func() { repoHead = old })
}

func stubRepoWorktree(t *testing.T, stub func(*git.Repository) (*git.Worktree, error)) {
	old := repoWorktree
	repoWorktree = stub
	t.Cleanup(func() { repoWorktree = old })
}

func stubWorktreeStatus(t *testing.T, stub func(*git.Worktree) (git.Status, error)) {
	old := worktreeStatus
	worktreeStatus = stub
	t.Cleanup(func() { worktreeStatus = old })
}

func stubFilepathAbs(t *testing.T, stub func(string) (string, error)) {
	old := filepathAbs
	filepathAbs = stub
	t.Cleanup(func() { filepathAbs = old })
}

func stubFilepathRel(t *testing.T, stub func(string, string) (string, error)) {
	old := filepathRel
	filepathRel = stub
	t.Cleanup(func() { filepathRel = old })
}

func stubWorktreeAdd(t *testing.T, stub func(*git.Worktree, string) (plumbing.Hash, error)) {
	old := worktreeAdd
	worktreeAdd = stub
	t.Cleanup(func() { worktreeAdd = old })
}

func stubIsCtxDone(t *testing.T, stub func(context.Context) bool) {
	old := isCtxDone
	isCtxDone = stub
	t.Cleanup(func() { isCtxDone = old })
}

func initRepoWithCommit(t *testing.T) (string, *git.Repository, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gitutils-coverage-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := worktree.Add("file.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	_, err = worktree.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "T", Email: "e", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	return tempDir, repo, filePath
}

func TestShortHash(t *testing.T) {
	if got := shortHash("abcd"); got != "abcd" {
		t.Fatalf("expected shortHash to keep short string, got %q", got)
	}
	if got := shortHash("abcdefg123"); got != "abcdefg" {
		t.Fatalf("expected shortHash to trim, got %q", got)
	}
}

func TestGetRepositoryRoot_AbsError(t *testing.T) {
	stubFilepathAbs(t, func(string) (string, error) {
		return "", errors.New("abs boom")
	})

	if got := GetRepositoryRoot("whatever"); got != "" {
		t.Fatalf("expected empty repo root on abs error, got %q", got)
	}
}

func TestGetDirStatus_HeadError(t *testing.T) {
	_, repo, _ := initRepoWithCommit(t)
	stubRepoHead(t, func(*git.Repository) (*plumbing.Reference, error) {
		return nil, errors.New("head boom")
	})

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	status := GetDirStatus(context.Background(), repo, wt.Filesystem.Root())
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Branch != "unknown" {
		t.Fatalf("expected branch unknown on head error, got %q", status.Branch)
	}
}

func TestGetDirStatus_HeadNil(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	stubRepoHead(t, func(*git.Repository) (*plumbing.Reference, error) {
		return nil, nil
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Branch != "unknown" {
		t.Fatalf("expected branch unknown when head is nil, got %q", status.Branch)
	}
}

func TestGetDirStatus_WorktreeErrorSecondCall(t *testing.T) {
	_, repo, _ := initRepoWithCommit(t)

	callCount := 0
	stubRepoWorktree(t, func(r *git.Repository) (*git.Worktree, error) {
		callCount++
		if callCount == 1 {
			return r.Worktree()
		}
		return nil, errors.New("worktree boom")
	})

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	status := GetDirStatus(context.Background(), repo, wt.Filesystem.Root())
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetDirStatus_WorktreeStatusError(t *testing.T) {
	_, repo, _ := initRepoWithCommit(t)
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return nil, errors.New("status boom")
	})

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	status := GetDirStatus(context.Background(), repo, wt.Filesystem.Root())
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetDirStatus_RelPathErrorAndSkipUnmodified(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	stubFilepathRel(t, func(string, string) (string, error) {
		return "", errors.New("rel boom")
	})
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			"a.txt": &git.FileStatus{Worktree: git.Unmodified, Staging: git.Unmodified},
			"b.txt": &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
		}, nil
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected 1 file changed, got %d", status.FilesChanged)
	}
}

func TestGetDirStatus_ContextCanceledInLoop(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	stubIsCtxDone(t, func(context.Context) bool {
		return true
	})
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			"b.txt": &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
		}, nil
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetFileStatus_WorktreeErrorSecondCall(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)

	callCount := 0
	stubRepoWorktree(t, func(r *git.Repository) (*git.Worktree, error) {
		callCount++
		if callCount == 1 {
			return r.Worktree()
		}
		return nil, errors.New("worktree boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetFileStatus_HeadError(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	stubRepoHead(t, func(*git.Repository) (*plumbing.Reference, error) {
		return nil, errors.New("head boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Branch != "unknown" {
		t.Fatalf("expected branch unknown on head error, got %q", status.Branch)
	}
}

func TestGetFileStatus_WorktreeStatusError(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return nil, errors.New("status boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetFileStatus_RelPathError_Stubbed(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	stubFilepathRel(t, func(string, string) (string, error) {
		return "", errors.New("rel boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("expected FilesChanged=0 on rel path error, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_UnmodifiedEntryInDirtyStatus(t *testing.T) {
	dir, repo, filePath := initRepoWithCommit(t)
	relPath, err := filepath.Rel(dir, filePath)
	if err != nil {
		t.Fatalf("failed to get rel path: %v", err)
	}
	relPath = filepath.ToSlash(relPath)

	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			relPath:   &git.FileStatus{Worktree: git.Unmodified, Staging: git.Unmodified},
			"other":   &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
			"other2":  &git.FileStatus{Worktree: git.Unmodified, Staging: git.Modified},
			"ignored": &git.FileStatus{Worktree: git.Unmodified, Staging: git.Unmodified},
		}, nil
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("expected FilesChanged=0 for unmodified entry, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_ContextCanceledInLoop(t *testing.T) {
	dir, repo, filePath := initRepoWithCommit(t)
	relPath, err := filepath.Rel(dir, filePath)
	if err != nil {
		t.Fatalf("failed to get rel path: %v", err)
	}
	relPath = filepath.ToSlash(relPath)

	stubIsCtxDone(t, func(context.Context) bool {
		return true
	})
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			relPath: &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
		}, nil
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestGetFileStatus_ContextCanceledAfterHead(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)

	ctx, cancel := context.WithCancel(context.Background())
	stubRepoHead(t, func(r *git.Repository) (*plumbing.Reference, error) {
		ref, err := r.Head()
		cancel()
		return ref, err
	})

	status := GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
}

func TestCanBeStaged_StatusMissingEntry(t *testing.T) {
	dir, _, _ := initRepoWithCommit(t)
	targetPath := filepath.Join(dir, "missing.txt")

	stubFilepathRel(t, func(string, string) (string, error) {
		return "not-in-map", nil
	})
	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			"other.txt": &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
		}, nil
	})

	can, err := CanBeStaged(targetPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if can {
		t.Fatal("expected CanBeStaged to be false when entry is missing")
	}
}

func TestCanBeStaged_WorktreeError(t *testing.T) {
	_, _, filePath := initRepoWithCommit(t)

	stubRepoWorktree(t, func(*git.Repository) (*git.Worktree, error) {
		return nil, errors.New("worktree boom")
	})

	can, err := CanBeStaged(filePath)
	if err == nil || can {
		t.Fatalf("expected error for worktree failure, got can=%v err=%v", can, err)
	}
}

func TestCanBeStaged_UnmodifiedEntryInDirtyStatus(t *testing.T) {
	dir, _, filePath := initRepoWithCommit(t)
	relPath, err := filepath.Rel(dir, filePath)
	if err != nil {
		t.Fatalf("failed to get rel path: %v", err)
	}
	relPath = filepath.ToSlash(relPath)

	stubWorktreeStatus(t, func(*git.Worktree) (git.Status, error) {
		return git.Status{
			relPath:  &git.FileStatus{Worktree: git.Unmodified, Staging: git.Unmodified},
			"other":  &git.FileStatus{Worktree: git.Modified, Staging: git.Unmodified},
			"other2": &git.FileStatus{Worktree: git.Unmodified, Staging: git.Modified},
		}, nil
	})

	can, err := CanBeStaged(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if can {
		t.Fatal("expected CanBeStaged to be false for unmodified entry")
	}
}

func TestGetWorktreeAndRelPath_AbsError(t *testing.T) {
	stubFilepathAbs(t, func(string) (string, error) {
		return "", errors.New("abs boom")
	})

	_, _, _, err := getWorktreeAndRelPath("whatever")
	if err == nil {
		t.Fatal("expected error for abs failure")
	}
}

func TestGetWorktreeAndRelPath_RelError(t *testing.T) {
	dir, _, filePath := initRepoWithCommit(t)

	stubFilepathRel(t, func(string, string) (string, error) {
		return "", errors.New("rel boom")
	})

	_, _, _, err := getWorktreeAndRelPath(filePath)
	if err == nil {
		t.Fatal("expected error for rel failure")
	}

	_ = dir
}

func TestIsCtxDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !isCtxDone(ctx) {
		t.Fatal("expected canceled context to return true")
	}
	if isCtxDone(context.Background()) {
		t.Fatal("expected background context to return false")
	}
}

func TestStageDir_AddError(t *testing.T) {
	dir, _, _ := initRepoWithCommit(t)
	targetDir := filepath.Join(dir, "stage_dir")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write target file: %v", err)
	}

	stubWorktreeAdd(t, func(*git.Worktree, string) (plumbing.Hash, error) {
		return plumbing.Hash{}, errors.New("add boom")
	})

	err := StageDir(targetDir, false)
	if err == nil {
		t.Fatal("expected error from worktree.Add")
	}
}

func TestStageDir_GetWorktreeAndRelPathError(t *testing.T) {
	stubFilepathAbs(t, func(string) (string, error) {
		return "", errors.New("abs boom")
	})

	err := StageDir("whatever", false)
	if err == nil {
		t.Fatal("expected error from getWorktreeAndRelPath")
	}
}
