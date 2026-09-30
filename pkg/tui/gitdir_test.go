package tui

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// fakeRepoStatus installs a repository at root whose work tree has status.
func fakeRepoStatus(t *testing.T, root string, status git.Status) {
	t.Helper()
	fakeGit(t, root, nil)
	oldWt, oldStat, oldRel, oldIgnore, oldIgnored := gitWorktreeOf, gitWorktreeStat, filepathRel, loadGitIgnore, isIgnoredInGit
	t.Cleanup(func() {
		gitWorktreeOf, gitWorktreeStat, filepathRel, loadGitIgnore, isIgnoredInGit = oldWt, oldStat, oldRel, oldIgnore, oldIgnored
	})
	gitWorktreeOf = func(*git.Repository) (*git.Worktree, error) { return &git.Worktree{}, nil }
	gitWorktreeStat = func(*git.Worktree) (git.Status, error) { return status, nil }
	loadGitIgnore = func(string) gitignore.Matcher { return nil }
	isIgnoredInGit = func(p string, _ gitignore.Matcher) bool { return p == "ignored.log" }
}

func TestLoadGitDirStatus(t *testing.T) {
	fakeRepoStatus(t, "/repo", git.Status{
		"b.go":        {Worktree: git.Modified, Staging: git.Unmodified},
		"a.go":        {Worktree: git.Untracked, Staging: git.Unmodified},
		"sub/c.go":    {Staging: git.Deleted},
		"sub/d.go":    {Staging: git.Modified, Worktree: git.Modified},
		"clean.go":    {Worktree: git.Unmodified, Staging: git.Unmodified},
		"ignored.log": {Worktree: git.Modified},
	})
	if loadGitDirStatus("/elsewhere") != nil {
		t.Fatal("outside a repository there is no status")
	}
	st := loadGitDirStatus("/repo")
	if st.Err != nil || st.RepoRoot != "/repo" || len(st.Entries) != 4 {
		t.Fatalf("every changed file that is not ignored: %+v", st)
	}
	if first := st.Entries[0]; first.DisplayName != "a.go" || first.Badge.Text != "A" || first.Staged || first.FullPath != "/repo/a.go" {
		t.Fatalf("entries are sorted: %+v", first)
	}
	sub := loadGitDirStatus("/repo/sub")
	if len(sub.Entries) != 2 || sub.Entries[0].DisplayName != "c.go" || !sub.Entries[0].Staged || sub.Entries[0].Badge.Text != "D" {
		t.Fatalf("a subdirectory shows its own files only: %+v", sub.Entries)
	}
	filepathRel = func(string, string) (string, error) { return "", errors.New("no relation") }
	if len(loadGitDirStatus("/repo/sub").Entries) != 4 {
		t.Fatal("without a relative path the whole repository shows")
	}
}

func TestLoadGitDirStatusFailures(t *testing.T) {
	fakeRepoStatus(t, "/repo", nil)
	gitOpen = func(string) (*git.Repository, error) { return nil, errors.New("open") }
	if st := loadGitDirStatus("/repo"); st.Err == nil || st.RepoRoot != "/repo" {
		t.Fatalf("open: %+v", st)
	}
	gitOpen = func(string) (*git.Repository, error) { return &git.Repository{}, nil }
	gitWorktreeOf = func(*git.Repository) (*git.Worktree, error) { return nil, errors.New("worktree") }
	if st := loadGitDirStatus("/repo"); st.Err == nil {
		t.Fatalf("worktree: %+v", st)
	}
	gitWorktreeOf = func(*git.Repository) (*git.Worktree, error) { return &git.Worktree{}, nil }
	gitWorktreeStat = func(*git.Worktree) (git.Status, error) { return nil, errors.New("status") }
	if st := loadGitDirStatus("/repo"); st.Err == nil {
		t.Fatalf("status: %+v", st)
	}
}

func TestBadgeForStatus(t *testing.T) {
	cases := []struct {
		status *git.FileStatus
		text   string
	}{
		{nil, "?"},
		{&git.FileStatus{Staging: git.Added}, "A"},
		{&git.FileStatus{Worktree: git.Untracked}, "A"},
		{&git.FileStatus{Staging: git.Deleted}, "D"},
		{&git.FileStatus{Worktree: git.Deleted}, "D"},
		{&git.FileStatus{Staging: git.Modified}, "M"},
		{&git.FileStatus{Worktree: git.Modified}, "M"},
		{&git.FileStatus{Staging: git.Renamed}, "M"},
		{&git.FileStatus{Worktree: git.Copied}, "M"},
		{&git.FileStatus{Worktree: git.UpdatedButUnmerged}, "?"},
	}
	for _, c := range cases {
		if got := badgeForStatus(c.status); got.Text != c.text || got.Label == "" {
			t.Errorf("%+v: %+v", c.status, got)
		}
	}
}

func TestStageCmd(t *testing.T) {
	fakeRepoStatus(t, "/repo", git.Status{"a.go": {Worktree: git.Modified}})
	oldStage, oldUnstage := stageInGit, unstageInGit
	defer func() { stageInGit, unstageInGit = oldStage, oldUnstage }()
	var staged, unstaged []string
	stageInGit = func(p string) error { staged = append(staged, p); return nil }
	unstageInGit = func(p string) error { unstaged = append(unstaged, p); return nil }

	msg := stageCmd("/repo", gitEntry{FullPath: "/repo/a.go"})().(gitDirLoadedMsg)
	if len(staged) != 1 || msg.Path != "/repo" || len(msg.Status.Entries) != 1 {
		t.Fatalf("stage: %v %+v", staged, msg)
	}
	stageCmd("/repo", gitEntry{FullPath: "/repo/a.go", Staged: true})()
	if len(unstaged) != 1 {
		t.Fatal("a staged file is unstaged")
	}
	stageInGit = func(string) error { return errors.New("locked") }
	msg = stageCmd("/repo", gitEntry{FullPath: "/repo/a.go"})().(gitDirLoadedMsg)
	if msg.Status.Err == nil {
		t.Fatal("the failure is part of the status")
	}
	msg = stageCmd("/elsewhere", gitEntry{})().(gitDirLoadedMsg)
	if msg.Status == nil || msg.Status.Err == nil {
		t.Fatal("a failure outside a repository still reports")
	}
	if gitDirCmd("/repo")().(gitDirLoadedMsg).Status == nil {
		t.Fatal("gitDirCmd reads the status")
	}
}
