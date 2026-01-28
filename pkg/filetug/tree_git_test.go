package filetug

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

func TestTree_SetDirContext_GitOptimization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tree-git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Initialize git repo
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	_ = os.Mkdir(subDir1, 0755)
	_ = os.Mkdir(subDir2, 0755)

	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = osfile.NewStore(tempDir)

	// Mock queueUpdateDraw to avoid hanging
	nav.queueUpdateDraw = func(f func()) {
		f()
	}

	tree := NewTree(nav)
	node := tview.NewTreeNode("root")

	dirEntries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}

	dirContext := files.NewDirContext(nav.store, tempDir, dirEntries)

	ctx := context.Background()
	tree.setDirContext(ctx, node, dirContext)

	// Give some time for goroutines to start and call updateGitStatus
	time.Sleep(100 * time.Millisecond)

	// Check if children were added
	children := node.GetChildren()
	if len(children) < 2 {
		t.Errorf("Expected at least 2 children (subdir1, subdir2), got %d", len(children))
	}
}

func TestNavigator_ShowDir_GitStatusText(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nav-git-text-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Initialize git repo
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a subdirectory
	subDirName := "subdir1"
	subDirPath := filepath.Join(tempDir, subDirName)
	_ = os.Mkdir(subDirPath, 0755)

	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = osfile.NewStore(tempDir)

	// Mock queueUpdateDraw to execute immediately
	nav.queueUpdateDraw = func(f func()) {
		f()
	}

	// Create a tree node for the subdirectory as it would be in the tree
	// In the tree, it would have a prefix like "📁subdir1"
	node := tview.NewTreeNode("📁" + subDirName).SetReference(subDirPath)

	ctx := context.Background()

	// When showDir is called (e.g., when a node is selected)
	nav.showDir(ctx, node, subDirPath, false)

	// Wait for goroutines
	time.Sleep(200 * time.Millisecond)

	text := node.GetText()
	if strings.Contains(text, tempDir) {
		t.Errorf("Node text contains full path, but it should only contain dir name and git status. Got: %q", text)
	}

	if !strings.HasPrefix(text, "📁"+subDirName) {
		t.Errorf("Node text should start with original name %q, got: %q", "📁"+subDirName, text)
	}
}

func TestNavigator_UpdateGitStatus_NoChanges(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-status-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// 1. Initialize git repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// 2. Create a clean subdirectory
	cleanSubDir := "clean-subdir"
	cleanSubDirPath := filepath.Join(tempDir, cleanSubDir)
	_ = os.Mkdir(cleanSubDirPath, 0755)

	// 3. Create a dirty subdirectory
	dirtySubDir := "dirty-subdir"
	dirtySubDirPath := filepath.Join(tempDir, dirtySubDir)
	_ = os.Mkdir(dirtySubDirPath, 0755)
	dirtyFile := filepath.Join(dirtySubDirPath, "file.txt")
	_ = os.WriteFile(dirtyFile, []byte("content"), 0644)

	app := tview.NewApplication()
	nav := NewNavigator(app)
	nav.store = osfile.NewStore(tempDir)
	nav.queueUpdateDraw = func(f func()) { f() }

	ctx := context.Background()

	t.Run("Root_Repo_Dir_Shows_Status_Even_If_Clean", func(t *testing.T) {
		node := tview.NewTreeNode("root").SetReference(tempDir)
		nav.updateGitStatus(ctx, repo, tempDir, node, "root")
		time.Sleep(50 * time.Millisecond)
		text := node.GetText()
		if !strings.Contains(text, "┆") {
			t.Errorf("Root repo directory should show git status even if clean, got: %q", text)
		}
	})

	t.Run("Clean_Subdirectory_Hides_Status", func(t *testing.T) {
		node := tview.NewTreeNode("clean").SetReference(cleanSubDirPath)
		nav.updateGitStatus(ctx, repo, cleanSubDirPath, node, "clean")
		time.Sleep(50 * time.Millisecond)
		text := node.GetText()
		if strings.Contains(text, "┆") {
			t.Errorf("Clean subdirectory should NOT show git status, got: %q", text)
		}
		if text != "clean" {
			t.Errorf("Expected node text to be 'clean', got: %q", text)
		}
	})

	t.Run("Dirty_Subdirectory_Shows_Status", func(t *testing.T) {
		node := tview.NewTreeNode("dirty").SetReference(dirtySubDirPath)
		nav.updateGitStatus(ctx, repo, dirtySubDirPath, node, "dirty")
		time.Sleep(50 * time.Millisecond)
		text := node.GetText()
		if !strings.Contains(text, "┆") {
			t.Errorf("Dirty subdirectory should show git status, got: %q", text)
		}
		if !strings.Contains(text, "ƒ1") {
			t.Errorf("Dirty subdirectory should show 1 file changed, got: %q", text)
		}
	})
}
