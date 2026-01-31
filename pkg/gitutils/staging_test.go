package gitutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestStaging(t *testing.T) {
	//t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-staging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// 1. Test CanBeStaged for non-git directory
	fileInNonGit := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(fileInNonGit, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	can, err := CanBeStaged(fileInNonGit)
	if err != nil {
		t.Errorf("CanBeStaged failed for non-git: %v", err)
	}
	if can {
		t.Error("CanBeStaged returned true for non-git directory")
	}

	// 2. Initialize git repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// 3. Test CanBeStaged for untracked file
	can, err = CanBeStaged(fileInNonGit)
	if err != nil {
		t.Errorf("CanBeStaged failed for untracked file: %v", err)
	}
	if !can {
		t.Error("Expected CanBeStaged to be true for untracked file")
	}

	// 4. Test StageFile
	err = StageFile(fileInNonGit)
	if err != nil {
		t.Errorf("StageFile failed: %v", err)
	}

	// 5. Verify it's staged
	worktree, _ := repo.Worktree()
	status, _ := worktree.Status()
	fileStatus := status.File("file.txt")
	if fileStatus.Staging == git.Unmodified {
		t.Error("File should be staged")
	}

	// 6. Test CanBeStaged for staged but unmodified in worktree
	// It should still return true because it has changes relative to HEAD (though here HEAD is empty)
	// Actually, go-git status for a new staged file shows Staging=Added, Worktree=Unmodified.
	can, err = CanBeStaged(fileInNonGit)
	if err != nil {
		t.Errorf("CanBeStaged failed for staged file: %v", err)
	}
	if !can {
		t.Error("Expected CanBeStaged to be true for staged file (Added)")
	}

	// 7. Commit and check
	_, err = worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	can, err = CanBeStaged(fileInNonGit)
	if err != nil {
		t.Errorf("CanBeStaged failed for clean file: %v", err)
	}
	if can {
		t.Error("Expected CanBeStaged to be false for clean file")
	}

	// 8. Modify and check
	if err := os.WriteFile(fileInNonGit, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	can, err = CanBeStaged(fileInNonGit)
	if err != nil {
		t.Errorf("CanBeStaged failed for modified file: %v", err)
	}
	if !can {
		t.Error("Expected CanBeStaged to be true for modified file")
	}

	// 9. Test UnstageFile
	// First stage the modified file
	if err := StageFile(fileInNonGit); err != nil {
		t.Fatalf("Failed to stage file for unstage test: %v", err)
	}

	// Verify it is staged
	status, _ = worktree.Status()
	fileStatus = status.File("file.txt")
	if fileStatus.Staging != git.Modified {
		t.Errorf("Expected status Staging=Modified, got %v", fileStatus.Staging)
	}

	// Now unstage it
	if err := UnstageFile(fileInNonGit); err != nil {
		t.Fatalf("UnstageFile failed: %v", err)
	}

	// Verify it is unstaged
	status, _ = worktree.Status()
	fileStatus = status.File("file.txt")
	if fileStatus.Staging != git.Unmodified {
		t.Errorf("Expected status Staging=Unmodified after unstage, got %v", fileStatus.Staging)
	}
	if fileStatus.Worktree != git.Modified {
		t.Errorf("Expected status Worktree=Modified after unstage, got %v", fileStatus.Worktree)
	}
}

func TestStagingErrors(t *testing.T) {
	//t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-staging-errors-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Test StageFile for non-existent file in non-git repo
	err = StageFile(filepath.Join(tempDir, "non-existent"))
	if err == nil {
		t.Error("Expected error for non-existent file in non-git repo")
	}

	// Init git repo
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Test StageFile for non-existent file in git repo
	err = StageFile(filepath.Join(tempDir, "non-existent"))
	if err == nil {
		t.Error("Expected error for non-existent file in git repo")
	}

	// Test UnstageFile for non-existent file
	err = UnstageFile(filepath.Join(tempDir, "non-existent"))
	if err == nil {
		t.Error("Expected error for non-existent file in UnstageFile")
	}

	// Test StageDir for non-existent dir
	err = StageDir(filepath.Join(tempDir, "non-existent-dir"), false)
	if err == nil {
		t.Error("Expected error for non-existent dir in StageDir")
	}

	// Test getWorktreeAndRelPath symlink coverage (macOS specific often, but good to try)
	// We'll just trigger the symlink logic if possible
	if realTempDir, err := filepath.EvalSymlinks(tempDir); err == nil && realTempDir != tempDir {
		// If we are already in a symlinked path, CanBeStaged will trigger EvalSymlinks
		_, _ = CanBeStaged(filepath.Join(tempDir, "file.txt"))
	}

	// Try to create a symlink to test EvalSymlinks explicitly
	linkDir := filepath.Join(os.TempDir(), "gitutils-link")
	_ = os.Remove(linkDir)
	if err := os.Symlink(tempDir, linkDir); err == nil {
		defer func() {
			_ = os.Remove(linkDir)
		}()
		_, _ = CanBeStaged(filepath.Join(linkDir, "file.txt"))
	}

	// Test CanBeStaged for a file that exists but is not in status (Clean)
	// file.txt was committed in TestStaging, but this is a fresh tempDir.
	cleanFile := filepath.Join(tempDir, "clean.txt")
	_ = os.WriteFile(cleanFile, []byte("clean"), 0644)
	w, _ := (func() (*git.Worktree, error) {
		r, _ := git.PlainOpen(tempDir)
		return r.Worktree()
	})()
	_, _ = w.Add("clean.txt")
	_, _ = w.Commit("clean", &git.CommitOptions{
		Author: &object.Signature{Name: "T", Email: "e", When: time.Now()},
	})

	can, _ := CanBeStaged(cleanFile)
	if can {
		t.Error("Expected CanBeStaged to be false for clean file")
	}

	// Test StageDir with recursive=true
	err = StageDir(tempDir, true)
	if err != nil {
		t.Errorf("Expected no error for StageDir(recursive=true), got %v", err)
	}

	// Test UnstageFile error (e.g. invalid path outside repo)
	err = UnstageFile("/invalid/path/far/away")
	if err == nil {
		t.Error("Expected error for UnstageFile on path outside repo")
	}

	// Test CanBeStaged for non-existent file in git repo
	can, err = CanBeStaged(filepath.Join(tempDir, "actually-missing.txt"))
	if can || err != nil {
		t.Errorf("Expected false and nil error for actually missing file in CanBeStaged (not in status), got %v, %v", can, err)
	}

	// Test CanBeStaged for a directory
	can, err = CanBeStaged(tempDir)
	if can || err != nil {
		t.Errorf("Expected false and nil error for directory in CanBeStaged (only files currently supported by its logic), got %v, %v", can, err)
	}

	// Test StageDir non-recursive
	err = StageDir(tempDir, false)
	if err != nil {
		t.Errorf("Expected no error for StageDir(recursive=false), got %v", err)
	}

	// Create a subdirectory with a file and test non-recursive StageDir on it
	subDir := filepath.Join(tempDir, "sub")
	_ = os.Mkdir(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("sub"), 0644)
	err = StageDir(subDir, false)
	if err != nil {
		t.Errorf("Expected no error for StageDir(recursive=false) on subDir, got %v", err)
	}

	// Test findRepoRoot for a file at the root of a repo
	rootFile := filepath.Join(tempDir, "root_file.txt")
	_ = os.WriteFile(rootFile, []byte("root"), 0644)
	can, _ = CanBeStaged(rootFile)
	// it can be staged because it's untracked
	if !can {
		t.Error("Expected CanBeStaged to be true for new root file")
	}

	// Test findRepoRoot for a path that doesn't exist but its parent does
	// It doesn't error because findRepoRoot goes up until it finds .git or root
	_, _ = CanBeStaged(filepath.Join(tempDir, "missing-dir", "file.txt"))

	// Test findRepoRoot for a path that cannot be statted (if possible, but hard to mock)

	// Test findRepoRoot starting from a file that is not in a git repo
	nonGitDir, _ := os.MkdirTemp("", "non-git-*")
	defer func() {
		_ = os.RemoveAll(nonGitDir)
	}()
	nonGitFile := filepath.Join(nonGitDir, "file.txt")
	_ = os.WriteFile(nonGitFile, []byte("test"), 0644)
	_, err = findRepoRoot(nonGitFile)
	if err == nil {
		t.Error("Expected error for findRepoRoot on non-git file")
	}
}

func TestStageDir(t *testing.T) {
	//t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-stagedir-test-*")
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
	worktree, _ := repo.Worktree()

	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	nestedDir := filepath.Join(subDir, "nested")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	file1 := filepath.Join(subDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("file1"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	file2 := filepath.Join(nestedDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("file2"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	t.Run("non-recursive", func(t *testing.T) {
		if err := StageDir(subDir, false); err != nil {
			t.Fatalf("StageDir non-recursive failed: %v", err)
		}

		status, _ := worktree.Status()
		if status.File("subdir/file1.txt").Staging == git.Unmodified {
			t.Error("subdir/file1.txt should be staged")
		}
		if status.File("subdir/nested/file2.txt").Staging != git.Untracked {
			t.Errorf("subdir/nested/file2.txt should NOT be staged, got %v", status.File("subdir/nested/file2.txt").Staging)
		}

		// Reset for next test
		_ = worktree.Reset(&git.ResetOptions{Mode: git.HardReset})
		// Clean untracked files if any (Reset Hard doesn't remove untracked)
		_ = os.WriteFile(file1, []byte("file1"), 0644)
		_ = os.WriteFile(file2, []byte("file2"), 0644)
	})

	t.Run("recursive", func(t *testing.T) {
		if err := StageDir(subDir, true); err != nil {
			t.Fatalf("StageDir recursive failed: %v", err)
		}

		status, _ := worktree.Status()
		if status.File("subdir/file1.txt").Staging == git.Unmodified {
			t.Error("subdir/file1.txt should be staged")
		}
		if status.File("subdir/nested/file2.txt").Staging == git.Unmodified {
			t.Error("subdir/nested/file2.txt should be staged")
		}
	})
}

func TestStageDir_ReadDirError(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-stagedir-readerr-*")
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
	err = os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	err = StageDir(filePath, false)
	if err == nil {
		t.Fatalf("Expected error when staging a file path as directory")
	}
}

func TestGetWorktreeAndRelPath_OpenRepoError(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-open-repo-error-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create fake .git dir: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, _, _, err = getWorktreeAndRelPath(filePath)
	if err == nil {
		t.Fatal("Expected error for fake git repo")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to open git repo") {
		t.Fatalf("Expected open repo error, got %v", err)
	}
}

func TestCanBeStaged_FakeRepoError(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-canbestaged-fake-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create fake .git dir: %v", err)
	}

	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	can, err := CanBeStaged(filePath)
	if err == nil {
		t.Fatalf("Expected error for fake git repo, got can=%v", can)
	}
}

func TestCanBeStaged_StatusError(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-canbestaged-statuserr-*")
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
	err = os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	indexPath := filepath.Join(tempDir, ".git", "index")
	err = os.MkdirAll(indexPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create index dir: %v", err)
	}

	_, err = CanBeStaged(filePath)
	if err == nil {
		t.Fatalf("Expected error when status cannot read index")
	}
}

func TestStageDir_RecursiveMissingPathError(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-stagedir-recursive-missing-*")
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

	missingPath := filepath.Join(tempDir, "missing-dir")
	err = StageDir(missingPath, true)
	if err == nil {
		t.Fatalf("Expected error for missing path with recursive StageDir")
	}
}
