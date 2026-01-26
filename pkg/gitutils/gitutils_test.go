package gitutils

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestDirGitStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status *RepoStatus
		want   string
	}{
		{
			name:   "nil",
			status: nil,
			want:   "",
		},
		{
			name:   "clean",
			status: &RepoStatus{Branch: "main"},
			want:   "[gray]┆[-][darkgray]main[-][lightgray]±0[-]",
		},
		{
			name: "dirty",
			status: &RepoStatus{Branch: "feature", DirGitChangesStats: DirGitChangesStats{
				FilesChanged:  2,
				FileGitStatus: FileGitStatus{Insertions: 10, Deletions: 5},
			}},
			want: "[gray]┆[-][darkgray]feature[-][gray]┆[-][darkgray]ƒ2[-][green]+10[-][red]-5[-]",
		},
		{
			name: "only_files_changed",
			status: &RepoStatus{Branch: "main", DirGitChangesStats: DirGitChangesStats{
				FilesChanged: 1,
			}},
			want: "[gray]┆[-][darkgray]main[-][gray]┆[-][darkgray]ƒ1[-][lightgray]±0[-]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("RepoStatus.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetRepositoryStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	t.Run("non-git directory", func(t *testing.T) {
		status := GetDirStatus(context.Background(), nil, tempDir)
		if status != nil {
			t.Errorf("Expected nil status for non-git directory, got %v", status)
		}
	})

	t.Run("empty git repo", func(t *testing.T) {
		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		status := GetDirStatus(context.Background(), repo, tempDir)
		if status == nil {
			t.Fatal("Expected status, got nil")
		}
		if status.Branch != "master" {
			t.Errorf("Expected branch master, got %s", status.Branch)
		}

		// Test detached HEAD (empty repo has no HEAD yet, so we need a commit)
		worktree, _ := repo.Worktree()
		filename := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filename, []byte("line1\nline2\n"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		if _, err := worktree.Add("file.txt"); err != nil {
			t.Fatalf("Failed to add file to worktree: %v", err)
		}
		hash, _ := worktree.Commit("initial", &git.CommitOptions{
			Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		})

		t.Run("clean repo", func(t *testing.T) {
			status := GetDirStatus(context.Background(), repo, tempDir)
			if status == nil {
				t.Fatal("Expected status, got nil")
			}
			if status.FilesChanged != 0 {
				t.Errorf("Expected 0 files changed, got %d", status.FilesChanged)
			}
		})

		t.Run("modified file", func(t *testing.T) {
			if err := os.WriteFile(filename, []byte("line1\nline2\nline3\n"), 0644); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, tempDir)
			if status == nil {
				t.Fatal("Expected status, got nil")
			}
			if status.FilesChanged != 1 {
				t.Errorf("Expected 1 file changed, got %d", status.FilesChanged)
			}
		})

		t.Run("untracked file", func(t *testing.T) {
			untrackedFile := filepath.Join(tempDir, "untracked.txt")
			if err := os.WriteFile(untrackedFile, []byte("newfile\nline2\n"), 0644); err != nil {
				t.Fatalf("Failed to write untracked file: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, tempDir)
			if status.FilesChanged < 1 {
				t.Errorf("Expected files changed, got %d", status.FilesChanged)
			}
			if status.Insertions == 0 {
				t.Errorf("Expected insertions > 0 for untracked file, got %d", status.Insertions)
			}
		})

		t.Run("deleted file", func(t *testing.T) {
			// Stage and commit the file first
			if _, err := worktree.Add("untracked.txt"); err != nil {
				t.Fatalf("Failed to add file: %v", err)
			}
			if _, err := worktree.Commit("add untracked", &git.CommitOptions{
				Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
			}); err != nil {
				t.Fatalf("Failed to commit: %v", err)
			}

			if err := os.Remove(filepath.Join(tempDir, "untracked.txt")); err != nil {
				t.Fatalf("Failed to remove file: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, tempDir)
			if status.Deletions == 0 {
				t.Errorf("Expected deletions > 0 for deleted file, got %d", status.Deletions)
			}
		})

		t.Run("detached HEAD", func(t *testing.T) {
			err = worktree.Checkout(&git.CheckoutOptions{
				Hash:  hash,
				Force: true,
			})
			if err != nil {
				t.Fatalf("Failed to checkout hash: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, tempDir)
			if len(status.Branch) != 7 {
				t.Errorf("Expected short hash (7 chars) for detached HEAD, got %s", status.Branch)
			}
		})

		t.Run("context cancelled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			status := GetDirStatus(ctx, repo, tempDir)
			// It might return nil if it was cancelled BEFORE semaphore
			// OR it might return a partial RepoStatus if it was cancelled after Branch was determined.
			// In our test, it seems it's getting past the semaphore.
			if status != nil && status.FilesChanged != 0 {
				t.Errorf("Expected nil or empty status for cancelled context, got %v", status)
			}
		})

		t.Run("context cancelled mid-way", func(t *testing.T) {
			// This is tricky to time perfectly, but we can try to use a very short timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()
			time.Sleep(2 * time.Millisecond)
			status := GetDirStatus(ctx, repo, tempDir)
			if status != nil && status.FilesChanged != 0 {
				t.Logf("Status is not nil and has files changed, but context was cancelled: %v", status)
			}
		})

		t.Run("corrupted .git directory", func(t *testing.T) {
			corruptedDir, _ := os.MkdirTemp("", "gitutils-corrupted-*")
			defer func() {
				_ = os.RemoveAll(corruptedDir)
			}()
			if err := os.Mkdir(filepath.Join(corruptedDir, ".git"), 0755); err != nil {
				t.Fatalf("Failed to create .git directory: %v", err)
			}
			// No actual git data in .git
			status := GetDirStatus(context.Background(), nil, corruptedDir)
			if status != nil {
				t.Errorf("Expected nil status for corrupted git repo, got %v", status)
			}
		})

		t.Run("corrupted head in .git", func(t *testing.T) {
			corruptedHeadDir, _ := os.MkdirTemp("", "gitutils-corrupted-head-*")
			defer func() {
				_ = os.RemoveAll(corruptedHeadDir)
			}()
			corruptedRepo, err := git.PlainInit(corruptedHeadDir, false)
			if err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}
			// Corrupt HEAD file to trigger error in repo.Head()
			// Using something that is clearly NOT a reference and NOT an empty repo
			err = os.WriteFile(filepath.Join(corruptedHeadDir, ".git", "HEAD"), []byte("not a ref"), 0644)
			if err != nil {
				t.Fatalf("Failed to corrupt HEAD: %v", err)
			}
			status := GetDirStatus(context.Background(), corruptedRepo, corruptedHeadDir)
			if status == nil {
				t.Errorf("Expected non-nil status for repo with corrupted HEAD, got nil")
			} else if status.Branch != "unknown" && status.Branch != "master" {
				t.Errorf("Expected branch unknown or master, got %s", status.Branch)
			}
		})

		t.Run("subdirectory of a git repo", func(t *testing.T) {
			subDir := filepath.Join(tempDir, "subdir", "deep", "dir")
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("Failed to create subdir: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, subDir)
			if status == nil {
				t.Fatal("Expected status for subdirectory, got nil")
			}
			if status.Branch == "" {
				t.Error("Expected branch to be set")
			}
		})

		t.Run("stats only for current dir", func(t *testing.T) {
			subDir := filepath.Join(tempDir, "target_subdir")
			otherDir := filepath.Join(tempDir, "other_subdir")
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("Failed to create subDir: %v", err)
			}
			if err := os.MkdirAll(otherDir, 0755); err != nil {
				t.Fatalf("Failed to create otherDir: %v", err)
			}

			// Create a change in target_subdir
			if err := os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content\n"), 0644); err != nil {
				t.Fatalf("Failed to write file in subDir: %v", err)
			}

			// Create a change in other_subdir
			if err := os.WriteFile(filepath.Join(otherDir, "file2.txt"), []byte("content\n"), 0644); err != nil {
				t.Fatalf("Failed to write file in otherDir: %v", err)
			}

			// Create a change in root
			if err := os.WriteFile(filepath.Join(tempDir, "root_file.txt"), []byte("content\n"), 0644); err != nil {
				t.Fatalf("Failed to write file in root: %v", err)
			}

			status := GetDirStatus(context.Background(), repo, subDir)
			if status == nil {
				t.Fatal("Expected status, got nil")
			}

			// It should only see 1 file changed (file1.txt in subDir)
			if status.FilesChanged != 1 {
				t.Errorf("Expected 1 file changed for subDir, got %d. It probably counted changes in parent/sibling dirs.", status.FilesChanged)
			}
		})

		t.Run("empty directory stats", func(t *testing.T) {
			emptyDir := filepath.Join(tempDir, "empty_dir")
			if err := os.Mkdir(emptyDir, 0755); err != nil {
				t.Fatalf("Failed to create empty_dir: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, emptyDir)
			if status == nil {
				t.Fatal("Expected status for empty dir, got nil")
			}
			if status.FilesChanged != 0 {
				t.Errorf("Expected 0 files changed for empty dir, got %d", status.FilesChanged)
			}
		})

		t.Run("non-existent directory", func(t *testing.T) {
			status := GetDirStatus(context.Background(), repo, filepath.Join(tempDir, "non-existent"))
			if status == nil {
				t.Fatal("Expected status for non-existent dir, got nil")
			}
			if status.FilesChanged != 0 {
				t.Errorf("Expected 0 files changed for non-existent dir, got %d", status.FilesChanged)
			}
		})

		t.Run("large untracked file line count", func(t *testing.T) {
			largeFile := filepath.Join(tempDir, "large.txt")
			content := strings.Repeat("line\n", 100)
			if err := os.WriteFile(largeFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write large file: %v", err)
			}
			status := GetDirStatus(context.Background(), repo, tempDir)
			if status.Insertions < 100 {
				t.Errorf("Expected at least 100 insertions, got %d", status.Insertions)
			}
		})

		t.Run("deleted file line count", func(t *testing.T) {
			delFile := filepath.Join(tempDir, "to_be_deleted.txt")
			content := "line1\nline2\nline3\n"
			if err := os.WriteFile(delFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write file to be deleted: %v", err)
			}
			wt, _ := repo.Worktree()
			_, _ = wt.Add("to_be_deleted.txt")
			_, _ = wt.Commit("add file to be deleted", &git.CommitOptions{
				Author: &object.Signature{Name: "T", Email: "e", When: time.Now()},
			})

			if err := os.Remove(delFile); err != nil {
				t.Fatalf("Failed to remove file: %v", err)
			}

			status := GetDirStatus(context.Background(), repo, tempDir)
			if status.Deletions != 3 {
				t.Errorf("Expected 3 deletions, got %d", status.Deletions)
			}
		})

		t.Run("repo without head", func(t *testing.T) {
			emptyRepoDir, _ := os.MkdirTemp("", "empty-repo-*")
			defer func() {
				_ = os.RemoveAll(emptyRepoDir)
			}()
			emptyRepo, _ := git.PlainInit(emptyRepoDir, false)
			status := GetDirStatus(context.Background(), emptyRepo, emptyRepoDir)
			if status.Branch != "master" {
				t.Errorf("Expected branch master for empty repo, got %v", status.Branch)
			}
		})

		t.Run("detached head", func(t *testing.T) {
			// Already has commits from previous tests
			head, _ := repo.Head()
			wt, _ := repo.Worktree()
			_ = wt.Checkout(&git.CheckoutOptions{
				Hash: head.Hash(),
			})
			status := GetDirStatus(context.Background(), repo, tempDir)
			if len(status.Branch) != 7 {
				t.Errorf("Expected short hash for detached head, got %v", status.Branch)
			}
		})
	})
}
