package tui

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runWorktreeTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newWorktreeTestRepository(t *testing.T) (canonical, linked string) {
	t.Helper()
	root := t.TempDir()
	canonical = filepath.Join(root, "repo")
	linked = filepath.Join(root, "linked")
	if err := os.Mkdir(canonical, 0o755); err != nil {
		t.Fatal(err)
	}
	runWorktreeTestGit(t, canonical, "init", "-b", "main")
	runWorktreeTestGit(t, canonical, "config", "user.email", "filetug@example.test")
	runWorktreeTestGit(t, canonical, "config", "user.name", "FileTug Test")
	if err := os.WriteFile(filepath.Join(canonical, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runWorktreeTestGit(t, canonical, "add", "README.md")
	runWorktreeTestGit(t, canonical, "commit", "-m", "initial")
	runWorktreeTestGit(t, canonical, "worktree", "add", "-b", "feature", linked)
	canonical, _ = filepath.EvalSymlinks(canonical)
	linked, _ = filepath.EvalSymlinks(linked)
	return canonical, linked
}

func installFakeWB(t *testing.T, output string, exitCode int) {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' '" + strings.ReplaceAll(output, "'", "'\\''") + "'\n"
	if exitCode != 0 {
		script += "exit " + string(rune('0'+exitCode)) + "\n"
	}
	path := filepath.Join(binDir, "wb")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestParseGitWorktrees(t *testing.T) {
	canonical := filepath.Clean("/tmp/canonical")
	linked := filepath.Clean("/tmp/linked")
	output := "worktree " + linked + "\nHEAD 1234567890abcdef\ndetached\nlocked reason\nprunable reason\n\n" +
		"worktree " + canonical + "\nHEAD abcdef1234567890\nbranch refs/heads/main\n\n"
	items, err := parseGitWorktrees(output, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !items[0].Canonical || items[0].Branch != "main" {
		t.Fatalf("unexpected parsed worktrees: %+v", items)
	}
	if !items[1].Detached || !items[1].Locked || !items[1].Prunable {
		t.Fatalf("linked state not parsed: %+v", items[1])
	}
	if _, err := parseGitWorktrees("", canonical); err == nil {
		t.Fatal("expected empty worktree report to fail")
	}
}

func TestGitWorktreeInventory(t *testing.T) {
	canonical, linked := newWorktreeTestRepository(t)
	resolved, items, err := readGitWorktrees(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != canonical || len(items) != 2 || items[1].Path != linked {
		t.Fatalf("unexpected inventory: canonical=%s items=%+v", resolved, items)
	}

	enrichGitWorktrees(context.Background(), items)
	for _, item := range items {
		if !item.StatusRead || item.LastCommit.IsZero() {
			t.Fatalf("worktree was not enriched: %+v", item)
		}
	}
	items = append(items, worktreeInfo{Path: filepath.Join(t.TempDir(), "missing")}, worktreeInfo{Prunable: true})
	enrichGitWorktrees(context.Background(), items)

	if got := gitRepositoryRoot(linked); got != linked {
		t.Fatalf("linked root = %q", got)
	}
	if got := gitRepositoryRoot(t.TempDir()); got != "" {
		t.Fatalf("non-repository root = %q", got)
	}
	if _, err := canonicalWorktreePath(context.Background(), t.TempDir()); err == nil {
		t.Fatal("expected a non-repository to fail canonical resolution")
	}

	bare := filepath.Join(t.TempDir(), "bare.git")
	if out, err := exec.Command("git", "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}
	if _, err := canonicalWorktreePath(context.Background(), bare); err == nil {
		t.Fatal("expected bare repository common directory to be unsupported")
	}
}

func TestWBWorktreeMetadata(t *testing.T) {
	canonical := filepath.Join(t.TempDir(), "org", "repo")
	managedPath := filepath.Join(t.TempDir(), "managed")
	report := wbOrphanReport{Families: []wbWorktreeFamily{{Worktrees: []wbWorktreeInfo{{
		Path: managedPath, CanonicalDir: canonical, HasManifest: true, EffortID: "effort",
	}}}}}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	installFakeWB(t, string(encoded), 0)
	metadata, status := readWBWorktreeMetadata(context.Background(), canonical)
	if status != "WB metadata loaded" || !metadata[filepath.Clean(managedPath)].HasManifest {
		t.Fatalf("metadata=%+v status=%q", metadata, status)
	}

	otherCanonical := canonical + "-other"
	metadata, _ = readWBWorktreeMetadata(context.Background(), otherCanonical)
	if len(metadata) != 0 {
		t.Fatalf("unexpected metadata for another canonical: %+v", metadata)
	}
}

func TestWBWorktreeMetadataFallbacks(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		metadata, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if len(metadata) != 0 || status != "WB not installed; showing raw Git" {
			t.Fatalf("metadata=%+v status=%q", metadata, status)
		}
	})
	t.Run("command fails", func(t *testing.T) {
		installFakeWB(t, "", 1)
		_, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if status != "WB metadata unavailable" {
			t.Fatalf("status=%q", status)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		installFakeWB(t, "not-json", 0)
		_, status := readWBWorktreeMetadata(context.Background(), "/tmp/org/repo")
		if status != "WB returned unsupported JSON" {
			t.Fatalf("status=%q", status)
		}
	})
}

func TestWorktreePresentationHelpers(t *testing.T) {
	items := []worktreeInfo{{Branch: "z"}, {Branch: "a"}, {Branch: "main", Canonical: true}}
	sortWorktrees(items)
	if !items[0].Canonical || items[1].Branch != "a" {
		t.Fatalf("unexpected sort: %+v", items)
	}
	if got := repositoryLabel("/tmp/org/repo"); got != "org/repo" {
		t.Fatalf("label=%q", got)
	}
	if got := shortSHA("1234567890abcdef"); got != "1234567890" {
		t.Fatalf("short SHA=%q", got)
	}
	if got := shortSHA("short"); got != "short" {
		t.Fatalf("short SHA=%q", got)
	}
}

func TestReadGitWorktreesFailures(t *testing.T) {
	oldCommandContext := worktreeExecCommandContext
	t.Cleanup(func() { worktreeExecCommandContext = oldCommandContext })

	t.Run("canonical", func(t *testing.T) {
		worktreeExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "sh", "-c", "exit 1")
		}
		if _, _, err := readGitWorktrees(context.Background(), "/tmp/repo"); err == nil {
			t.Fatal("expected canonical resolution to fail")
		}
	})

	for _, test := range []struct {
		name   string
		script string
	}{
		{name: "list", script: "exit 1"},
		{name: "empty", script: "printf ''"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			worktreeExecCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				calls++
				if calls == 1 {
					return exec.CommandContext(ctx, "sh", "-c", "printf '/tmp/repo/.git'")
				}
				return exec.CommandContext(ctx, "sh", "-c", test.script)
			}
			if _, _, err := readGitWorktrees(context.Background(), "/tmp/repo"); err == nil {
				t.Fatal("expected worktree inventory to fail")
			}
		})
	}
}
