package gitutils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestFileGitStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status FileGitStatus
		want   string
	}{
		{"zero", FileGitStatus{0, 0}, "[lightgray]±0[-]"},
		{"insertions only", FileGitStatus{5, 0}, "[green]+5[-]"},
		{"deletions only", FileGitStatus{0, 3}, "[red]-3[-]"},
		{"both", FileGitStatus{5, 3}, "[green]+5[-][red]-3[-]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("FileGitStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
		status := GetRepositoryStatus(context.Background(), tempDir)
		if status != nil {
			t.Errorf("Expected nil status for non-git directory, got %v", status)
		}
	})

	t.Run("empty git repo", func(t *testing.T) {
		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		status := GetRepositoryStatus(context.Background(), tempDir)
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
			status := GetRepositoryStatus(context.Background(), tempDir)
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
			status := GetRepositoryStatus(context.Background(), tempDir)
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
			status := GetRepositoryStatus(context.Background(), tempDir)
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
			status := GetRepositoryStatus(context.Background(), tempDir)
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
			status := GetRepositoryStatus(context.Background(), tempDir)
			if len(status.Branch) != 7 {
				t.Errorf("Expected short hash (7 chars) for detached HEAD, got %s", status.Branch)
			}
		})

		t.Run("context cancelled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			status := GetRepositoryStatus(ctx, tempDir)
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
			status := GetRepositoryStatus(ctx, tempDir)
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
			status := GetRepositoryStatus(context.Background(), corruptedDir)
			if status != nil {
				t.Errorf("Expected nil status for corrupted git repo, got %v", status)
			}
		})

		t.Run("corrupted head in .git", func(t *testing.T) {
			corruptedHeadDir, _ := os.MkdirTemp("", "gitutils-corrupted-head-*")
			defer func() {
				_ = os.RemoveAll(corruptedHeadDir)
			}()
			_, err := git.PlainInit(corruptedHeadDir, false)
			if err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}
			// Corrupt HEAD file to trigger error in repo.Head()
			// Using something that is clearly NOT a reference and NOT an empty repo
			err = os.WriteFile(filepath.Join(corruptedHeadDir, ".git", "HEAD"), []byte("not a ref"), 0644)
			if err != nil {
				t.Fatalf("Failed to corrupt HEAD: %v", err)
			}
			status := GetRepositoryStatus(context.Background(), corruptedHeadDir)
			if status == nil {
				t.Errorf("Expected non-nil status for repo with corrupted HEAD, got nil")
			} else if status.Branch != "unknown" && status.Branch != "master" {
				t.Errorf("Expected branch unknown or master, got %s", status.Branch)
			}
		})
	})
}
