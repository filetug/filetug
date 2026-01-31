package gitutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRepositoryRoot(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "test_git_repo")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	repoRoot := filepath.Join(tempDir, "repo")
	err = os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(repoRoot, "a", "b", "c")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	nonRepoDir := filepath.Join(tempDir, "not_a_repo")
	err = os.MkdirAll(nonRepoDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		dirPath  string
		expected string
	}{
		{
			name:     "repo_root",
			dirPath:  repoRoot,
			expected: repoRoot,
		},
		{
			name:     "sub_dir",
			dirPath:  subDir,
			expected: repoRoot,
		},
		{
			name:     "non_repo",
			dirPath:  nonRepoDir,
			expected: "",
		},
		{
			name:     "file_in_repo",
			dirPath:  filepath.Join(repoRoot, "a", "file.txt"),
			expected: repoRoot,
		},
	}

	// Create a file in repo
	if err := os.WriteFile(filepath.Join(repoRoot, "a", "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRepositoryRoot(tt.dirPath)
			if got != tt.expected {
				t.Errorf("GetRepositoryRoot() = %v, want %v", got, tt.expected)
			}
		})
	}
}
