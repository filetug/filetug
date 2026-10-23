package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/uitest"
)

// fakeGit replaces the git seams: dir is inside a repository rooted at root,
// and statuses gives the status of each path.
func fakeGit(t *testing.T, root string, statuses map[string]*gitutils.RepoStatus) {
	t.Helper()
	oldRoot, oldOpen, oldDir, oldFile := gitRepoRoot, gitOpen, gitDirState, gitFileStat
	t.Cleanup(func() { gitRepoRoot, gitOpen, gitDirState, gitFileStat = oldRoot, oldOpen, oldDir, oldFile })
	gitRepoRoot = func(p string) string {
		if strings.HasPrefix(p, root) {
			return root
		}
		return ""
	}
	gitOpen = func(string) (*git.Repository, error) { return &git.Repository{}, nil }
	status := func(_ context.Context, _ *git.Repository, p string) *gitutils.RepoStatus { return statuses[p] }
	gitDirState, gitFileStat = status, status
}

func changes(branch string, files, added, removed int) *gitutils.RepoStatus {
	return &gitutils.RepoStatus{Branch: branch, DirGitChangesStats: gitutils.DirGitChangesStats{
		FilesChanged: files, FileGitStatus: gitutils.FileGitStatus{Insertions: added, Deletions: removed},
	}}
}

func collect(t *testing.T, first func() (msg any)) []gitStatusMsg {
	t.Helper()
	var out []gitStatusMsg
	msg := first()
	for msg != nil {
		m, ok := msg.(gitStatusMsg)
		if !ok {
			t.Fatalf("unexpected message %T", msg)
		}
		out = append(out, m)
		next := waitGit(m.stream)
		msg = next()
	}
	return out
}

func TestGitStatusCmdStreamsEveryChangedPath(t *testing.T) {
	fakeGit(t, "/repo", map[string]*gitutils.RepoStatus{
		"/repo/a":    changes("main", 1, 2, 0),
		"/repo/b":    changes("main", 0, 0, 0), // clean, not the root: no label
		"/repo/c":    changes("main", 3, 0, 4),
		"/repo/f.go": changes("main", 1, 1, 1),
	})
	reqs := []gitReq{{"/repo/a", true}, {"/repo/b", true}, {"/repo/c", true}, {"/repo/f.go", false}, {"/repo/none", true}}
	got := collect(t, func() any { return gitStatusCmd(context.Background(), 9, gitTree, "/repo", reqs)() })
	paths := map[string]bool{}
	for _, m := range got {
		paths[m.Path] = true
		if m.Seq != 9 || m.Scope != gitTree || m.Text == "" {
			t.Fatalf("%+v", m)
		}
	}
	if len(got) != 3 || !paths["/repo/a"] || !paths["/repo/c"] || !paths["/repo/f.go"] {
		t.Fatalf("only changed paths are reported: %v", paths)
	}
}

func TestGitStatusCmdEndsImmediately(t *testing.T) {
	fakeGit(t, "/repo", nil)
	if gitStatusCmd(context.Background(), 1, gitFiles, "/elsewhere", []gitReq{{"/x", true}})() != nil {
		t.Fatal("not a repository")
	}
	if gitStatusCmd(context.Background(), 1, gitFiles, "/repo", nil)() != nil {
		t.Fatal("nothing to read")
	}
	gitOpen = func(string) (*git.Repository, error) { return nil, errors.New("corrupt") }
	if gitStatusCmd(context.Background(), 1, gitFiles, "/repo", []gitReq{{"/repo/x", true}})() != nil {
		t.Fatal("a repository that cannot be opened")
	}
}

func TestGitStatusCmdStopsWhenCanceled(t *testing.T) {
	fakeGit(t, "/repo", map[string]*gitutils.RepoStatus{"/repo/a": changes("main", 1, 1, 0)})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if gitStatusCmd(ctx, 1, gitFiles, "/repo", []gitReq{{"/repo/a", true}})() != nil {
		t.Fatal("a canceled read reports nothing")
	}
}

func TestGitLabel(t *testing.T) {
	fakeGit(t, "/repo", map[string]*gitutils.RepoStatus{
		"/repo":     changes("main", 0, 0, 0),
		"/repo/sub": changes("main", 0, 0, 0),
		"/repo/f":   changes("main", 0, 0, 0),
	})
	ctx := context.Background()
	repo := &git.Repository{}
	if gitLabel(ctx, repo, gitReq{"/repo/missing", true}) != "" {
		t.Fatal("no status")
	}
	if gitLabel(ctx, repo, gitReq{"/repo/sub", true}) != "" || gitLabel(ctx, repo, gitReq{"/repo/f", false}) != "" {
		t.Fatal("clean paths have no label")
	}
	if got := uitest.Plain(gitLabel(ctx, repo, gitReq{"/repo", true})); got != "┆main±0" {
		t.Fatalf("the repository root is always labelled: %q", got)
	}
	canceled, cancel := context.WithCancel(ctx)
	gitDirState = func(context.Context, *git.Repository, string) *gitutils.RepoStatus {
		cancel()
		return changes("main", 1, 1, 0)
	}
	if gitLabel(canceled, repo, gitReq{"/repo/x", true}) != "" {
		t.Fatal("a status read after cancellation is dropped")
	}
}

func TestIsRepoRoot(t *testing.T) {
	fakeGit(t, "/repo", nil)
	if !isRepoRoot("/repo") || !isRepoRoot("/repo/") || isRepoRoot("/repo/sub") || isRepoRoot("/other") {
		t.Fatal("only the top of a repository")
	}
}

func TestFormatGitStatus(t *testing.T) {
	defer theme.SetDark(theme.Dark)
	for _, dark := range []bool{true, false} {
		theme.SetDark(dark)
		if got := uitest.Plain(formatGitStatus(&gitutils.RepoStatus{Branch: "main"})); got != "┆main±0" {
			t.Fatalf("no changes: %q", got)
		}
		if got := uitest.Plain(formatGitStatus(changes("dev", 3, 5, 2))); got != "┆dev┆ƒ3+5-2" {
			t.Fatalf("changes: %q", got)
		}
		if got := uitest.Plain(formatGitStatus(changes("dev", 1, 0, 2))); got != "┆dev┆ƒ1-2" {
			t.Fatalf("removals only: %q", got)
		}
	}
	if adaptive("#ffffff", "#000000") == nil {
		t.Fatal("a colour")
	}
}

func TestGitStatusFromFixtureRepository(t *testing.T) {
	// One real repository, no seams: the work tree of this test is not a
	// repository, so nothing is labelled and the stream ends.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "f"), "x")
	if gitStatusCmd(context.Background(), 1, gitFiles, dir, []gitReq{{filepath.Join(dir, "f"), false}})() != nil {
		t.Fatal("a plain directory has no git status")
	}
}
