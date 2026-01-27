package gitutils

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetDirStatus_UntrackedReadError(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	untrackedPath := filepath.Join(dir, "untracked.txt")
	if err := os.WriteFile(untrackedPath, []byte("line1\n"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	stubReadLimitedContent(t, func(_ io.Reader) (string, error) {
		return "", errors.New("read boom")
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Insertions != 0 {
		t.Fatalf("expected Insertions=0 on read error, got %d", status.Insertions)
	}
}

func TestGetDirStatus_HeadFileContentsError(t *testing.T) {
	dir, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	stubReadHeadFileContents(t, func(*object.File) (string, error) {
		return "", errors.New("contents boom")
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Insertions != 0 || status.Deletions != 0 {
		t.Fatalf("expected no insertions or deletions on contents error, got +%d -%d", status.Insertions, status.Deletions)
	}
}

func TestGetDirStatus_StagedNewFileCountsInsertions(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	addedPath := filepath.Join(dir, "added.txt")
	if err := os.WriteFile(addedPath, []byte("a\nb\n"), 0644); err != nil {
		t.Fatalf("failed to write added file: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := wt.Add("added.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Insertions != 2 {
		t.Fatalf("expected 2 insertions for staged file, got %d", status.Insertions)
	}
}

func TestGetFileStatus_StagedNewFileCountsInsertions(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	addedPath := filepath.Join(dir, "added.txt")
	if err := os.WriteFile(addedPath, []byte("a\nb\n"), 0644); err != nil {
		t.Fatalf("failed to write added file: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := wt.Add("added.txt"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	status := GetFileStatus(context.Background(), repo, addedPath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Insertions != 2 {
		t.Fatalf("expected 2 insertions for staged file, got %d", status.Insertions)
	}
}

func TestGetFileStatus_UntrackedReadError(t *testing.T) {
	dir, repo, _ := initRepoWithCommit(t)
	untrackedPath := filepath.Join(dir, "untracked.txt")
	if err := os.WriteFile(untrackedPath, []byte("line1\n"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	stubReadLimitedContent(t, func(_ io.Reader) (string, error) {
		return "", errors.New("read boom")
	})

	status := GetFileStatus(context.Background(), repo, untrackedPath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Insertions != 0 {
		t.Fatalf("expected Insertions=0 on read error, got %d", status.Insertions)
	}
}

func TestGetFileStatus_HeadFileContentsError(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	stubReadHeadFileContents(t, func(*object.File) (string, error) {
		return "", errors.New("contents boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Insertions != 0 || status.Deletions != 0 {
		t.Fatalf("expected no insertions or deletions on contents error, got +%d -%d", status.Insertions, status.Deletions)
	}
}

func TestGetDirStatus_ModifiedFileOpenError(t *testing.T) {
	dir, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_ModifiedFileOpenError(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
}

func TestGetDirStatus_ModifiedFileReadError(t *testing.T) {
	dir, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	stubReadLimitedContent(t, func(_ io.Reader) (string, error) {
		return "", errors.New("read boom")
	})

	status := GetDirStatus(context.Background(), repo, dir)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_ModifiedFileReadError(t *testing.T) {
	_, repo, filePath := initRepoWithCommit(t)
	if err := os.WriteFile(filePath, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	stubReadLimitedContent(t, func(_ io.Reader) (string, error) {
		return "", errors.New("read boom")
	})

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("expected FilesChanged=1, got %d", status.FilesChanged)
	}
}
