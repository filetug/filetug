package gitutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestGetDirStatus_Race(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "gitutils-race-test-*")
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

	// Create some subdirectories and files to make status work a bit
	for i := 0; i < 5; i++ {
		subDir := filepath.Join(tempDir, fmt.Sprintf("subdir%d", i))
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}
		for j := 0; j < 5; j++ {
			fileName := filepath.Join(subDir, fmt.Sprintf("file%d.txt", j))
			if err := os.WriteFile(fileName, []byte(fmt.Sprintf("content %d %d\n", i, j)), 0644); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}
		}
	}

	const goroutines = 50
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	ctx := context.Background()

	// Use multiple repositories to test the per-repo lock as well
	tempDir2, _ := os.MkdirTemp("", "gitutils-race-test2-*")
	defer func() {
		_ = os.RemoveAll(tempDir2)
	}()
	repo2, _ := git.PlainInit(tempDir2, false)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			r := repo
			d := tempDir
			if id%2 == 1 {
				r = repo2
				d = tempDir2
			}
			subDir := filepath.Join(d, fmt.Sprintf("subdir%d", id%5))
			for j := 0; j < iterations; j++ {
				_ = GetDirStatus(ctx, r, subDir)
			}
		}(i)
	}

	wg.Wait()
}
