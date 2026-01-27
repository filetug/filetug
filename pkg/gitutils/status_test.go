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

func TestGetFileStatus_UntrackedFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	repo, err := git.PlainOpen(tempDir)
	if err != nil {
		t.Fatalf("Failed to open git repo: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("Expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Branch == "" {
		t.Fatal("Expected non-empty branch")
	}
}

func TestGetFileStatus_UntrackedFileWithHead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-untracked-head-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	trackedPath := filepath.Join(tempDir, "tracked.txt")
	err = os.WriteFile(trackedPath, []byte("tracked\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write tracked file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = worktree.Add("tracked.txt")
	if err != nil {
		t.Fatalf("Failed to add tracked file: %v", err)
	}
	now := time.Now()
	signature := &object.Signature{Name: "T", Email: "e", When: now}
	_, err = worktree.Commit("commit tracked file", &git.CommitOptions{Author: signature})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, "untracked.txt")
	err = os.WriteFile(untrackedPath, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, untrackedPath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("Expected FilesChanged=1, got %d", status.FilesChanged)
	}
	if status.Insertions == 0 {
		t.Fatalf("Expected insertions for untracked file with head commit")
	}
}

func TestGetFileStatus_NilRepo(t *testing.T) {
	ctx := context.Background()
	status := GetFileStatus(ctx, nil, "file.txt")
	if status != nil {
		t.Fatalf("Expected nil status for nil repo, got %v", status)
	}
}

func TestGetFileStatus_BareRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-bare-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, true)
	if err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, tempDir)
	if status != nil {
		t.Fatalf("Expected nil status for bare repo, got %v", status)
	}
}

func TestGetFileStatus_CleanFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-clean-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	content := []byte("line1\nline2\n")
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = worktree.Add("file.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	now := time.Now()
	signature := &object.Signature{Name: "T", Email: "e", When: now}
	_, err = worktree.Commit("commit clean file", &git.CommitOptions{Author: signature})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	status := GetFileStatus(context.Background(), repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("Expected FilesChanged=0, got %d", status.FilesChanged)
	}
	if status.Insertions != 0 || status.Deletions != 0 {
		t.Fatalf("Expected no insertions or deletions, got +%d -%d", status.Insertions, status.Deletions)
	}
	if status.Branch == "" {
		t.Fatal("Expected non-empty branch")
	}
}

func TestGetFileStatus_ModifiedAndDeletedFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-moddel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	content := []byte("line1\nline2\n")
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = worktree.Add("file.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	now := time.Now()
	signature := &object.Signature{Name: "T", Email: "e", When: now}
	_, err = worktree.Commit("commit file", &git.CommitOptions{Author: signature})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	newContent := []byte("line1\nline2\nline3\n")
	err = os.WriteFile(filePath, newContent, 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 1 {
		t.Fatalf("Expected FilesChanged=1 for modified file, got %d", status.FilesChanged)
	}
	if status.Insertions != 1 || status.Deletions != 0 {
		t.Fatalf("Expected +1 -0 for modified file, got +%d -%d", status.Insertions, status.Deletions)
	}

	err = os.Remove(filePath)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	ctx = context.Background()
	status = GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status for deleted file")
	}
	if status.Deletions != 2 {
		t.Fatalf("Expected Deletions=2 for deleted file, got %d", status.Deletions)
	}
}

func TestGetFileStatus_FileNotInStatus(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-missing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, "untracked.txt")
	err = os.WriteFile(untrackedPath, []byte("content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	missingPath := filepath.Join(tempDir, "missing.txt")
	ctx := context.Background()
	status := GetFileStatus(ctx, repo, missingPath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("Expected FilesChanged=0 for missing file, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_RelPathError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-relerr-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	untrackedPath := filepath.Join(tempDir, "untracked.txt")
	err = os.WriteFile(untrackedPath, []byte("content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write untracked file: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, "")
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("Expected FilesChanged=0 for rel path error, got %d", status.FilesChanged)
	}
}

func TestGetFileStatus_ContextCanceled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-cancel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status := GetFileStatus(ctx, repo, filePath)
	if status != nil && status.FilesChanged != 0 {
		t.Fatalf("Expected nil or empty status for canceled context, got %v", status)
	}
}

func TestGetFileStatus_DetachedHeadAndCorruptedHead(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-heads-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = worktree.Add("file.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	now := time.Now()
	signature := &object.Signature{Name: "T", Email: "e", When: now}
	hash, err := worktree.Commit("commit for detached head", &git.CommitOptions{Author: signature})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	checkoutOpts := &git.CheckoutOptions{Hash: hash}
	err = worktree.Checkout(checkoutOpts)
	if err != nil {
		t.Fatalf("Failed to checkout detached head: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}
	if len(status.Branch) != 7 {
		t.Fatalf("Expected short hash branch for detached head, got %s", status.Branch)
	}

	headPath := filepath.Join(tempDir, ".git", "HEAD")
	err = os.WriteFile(headPath, []byte("not a ref"), 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt HEAD: %v", err)
	}

	ctx = context.Background()
	status = GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status for corrupted head")
	}
	if status.Branch != "unknown" && status.Branch != "master" {
		t.Fatalf("Expected branch unknown or master for corrupted head, got %s", status.Branch)
	}
}

func TestGetFileStatus_StatusError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitutils-file-status-statuserr-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	indexPath := filepath.Join(tempDir, ".git", "index")
	err = os.MkdirAll(indexPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create index dir: %v", err)
	}

	ctx := context.Background()
	status := GetFileStatus(ctx, repo, filePath)
	if status == nil {
		t.Fatal("Expected non-nil status when status fails")
	}
	if status.FilesChanged != 0 {
		t.Fatalf("Expected FilesChanged=0 when status fails, got %d", status.FilesChanged)
	}
}
