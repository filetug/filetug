package viewers

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/stretchr/testify/assert"
)

var gitDirStatusTestLock sync.Mutex

func TestGitDirStatusPreviewer_SetDirAndRefresh(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		close(started)
		<-release
		close(done)
		return gitDirStatusResult{}, nil
	}

	dirContext := files.NewDirContext(nil, "/tmp", nil)
	p.SetDir(dirContext, func(f func()) { f() })
	<-started
	cell := p.table.GetCell(0, 0)
	assert.Equal(t, "Loading...", cell.Text)

	close(release)
	<-done
	time.Sleep(10 * time.Millisecond)
}

func TestGitDirStatusPreviewer_Refresh_NoDirContext(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	p.refresh()
	cell := p.table.GetCell(0, 0)
	assert.Equal(t, "Not a git repository", cell.Text)
}

func TestGitDirStatusPreviewer_RefreshBranches(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	p.dirContext = files.NewDirContext(nil, "/repo", nil)

	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{}, nil
	}
	p.refresh()
	cell := p.table.GetCell(0, 0)
	assert.Equal(t, "Not a git repository", cell.Text)

	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{}, errors.New("load fail")
	}
	p.refresh()
	cell = p.table.GetCell(0, 0)
	assert.Equal(t, "load fail", cell.Text)

	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}
	p.refresh()
	cell = p.table.GetCell(0, 0)
	assert.Equal(t, "No changes", cell.Text)

	entry := gitDirStatusEntry{
		fullPath:    "/repo/file.txt",
		displayName: "file.txt",
		staged:      true,
		badge:       gitBadge{text: "A", color: tcell.ColorLightGreen, label: "added"},
	}
	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{repoRoot: "/repo", entries: []gitDirStatusEntry{entry}}, nil
	}
	p.refresh()
	cell = p.table.GetCell(0, 0)
	assert.Contains(t, cell.Text, "✓")
}

func TestGitDirStatusPreviewer_HandleInput(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	p.dirContext = files.NewDirContext(nil, "/repo", nil)

	p.entries = []gitDirStatusEntry{
		{fullPath: "/repo/file.txt", displayName: "file.txt", staged: false, badge: gitBadge{text: "A"}},
	}
	p.renderEntries()
	p.table.Select(0, 0)

	stageCalled := false
	p.stageFile = func(_ string) error {
		stageCalled = true
		return nil
	}
	stageDone := make(chan struct{})
	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		close(stageDone)
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}
	space := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	assert.Nil(t, p.handleInput(space))
	<-stageDone
	assert.True(t, stageCalled)

	p.entries = []gitDirStatusEntry{
		{fullPath: "/repo/file.txt", displayName: "file.txt", staged: true, badge: gitBadge{text: "A"}},
	}
	p.renderEntries()
	p.table.Select(0, 0)

	unstageCalled := false
	p.unstageFile = func(_ string) error {
		unstageCalled = true
		return nil
	}
	unstageDone := make(chan struct{})
	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		close(unstageDone)
		return gitDirStatusResult{repoRoot: "/repo"}, nil
	}
	assert.Nil(t, p.handleInput(space))
	<-unstageDone
	assert.True(t, unstageCalled)

	p.entries = []gitDirStatusEntry{
		{fullPath: "/repo/file.txt", displayName: "file.txt", staged: true, badge: gitBadge{text: "A"}},
	}
	p.renderEntries()
	p.table.Select(0, 0)
	p.unstageFile = func(_ string) error {
		return errors.New("unstage error")
	}
	assert.Nil(t, p.handleInput(space))
	cell := p.table.GetCell(0, 0)
	assert.Equal(t, "unstage error", cell.Text)

	p.entries = []gitDirStatusEntry{
		{fullPath: "/repo/file.txt", displayName: "file.txt", staged: false, badge: gitBadge{text: "A"}},
	}
	p.renderEntries()
	p.table.Select(0, 0)
	p.stageFile = func(_ string) error {
		return errors.New("stage error")
	}
	assert.Nil(t, p.handleInput(space))
	cell = p.table.GetCell(0, 0)
	assert.Equal(t, "stage error", cell.Text)

	p.entries = nil
	p.table.Select(1, 0)
	none := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	assert.Nil(t, p.handleInput(none))

	other := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	assert.Equal(t, other, p.handleInput(other))
}

func TestGitDirStatusPreviewer_Preview(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	p.statusLoader = func(_ string) (gitDirStatusResult, error) {
		return gitDirStatusResult{}, nil
	}
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "repo", isDir: true}, "/tmp")
	p.PreviewSingle(entry, nil, nil, func(f func()) { f() })
	cell := p.table.GetCell(0, 0)
	assert.Equal(t, "Loading...", cell.Text)
}

func TestGitDirStatusPreviewer_MainMeta(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	assert.Equal(t, p, p.Main())
	assert.Nil(t, p.Meta())
}

func TestLoadGitDirStatus_Seams(t *testing.T) {
	t.Parallel()
	gitDirStatusTestLock.Lock()
	t.Cleanup(gitDirStatusTestLock.Unlock)
	origPlainOpen := gitPlainOpen
	origRepoWorktree := repoWorktree
	origStatus := worktreeStatus
	origRel := filepathRel
	origRoot := getRepositoryRoot
	origFromSlash := filepathFromSlashFn
	origLoadGlobalIgnore := loadGlobalIgnore
	origIsIgnoredPath := isIgnoredPath

	defer func() {
		gitPlainOpen = origPlainOpen
		repoWorktree = origRepoWorktree
		worktreeStatus = origStatus
		filepathRel = origRel
		getRepositoryRoot = origRoot
		filepathFromSlashFn = origFromSlash
		loadGlobalIgnore = origLoadGlobalIgnore
		isIgnoredPath = origIsIgnoredPath
	}()

	loadGlobalIgnore = func(_ string) gitignore.Matcher {
		return nil
	}
	isIgnoredPath = func(_ string, _ gitignore.Matcher) bool {
		return false
	}

	getRepositoryRoot = func(_ string) string {
		return ""
	}
	result, err := loadGitDirStatus("/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "", result.repoRoot)

	getRepositoryRoot = func(_ string) string {
		return "/repo"
	}
	gitPlainOpen = func(_ string) (*git.Repository, error) {
		return nil, errors.New("open err")
	}
	result, err = loadGitDirStatus("/repo")
	assert.Error(t, err)
	assert.Equal(t, "/repo", result.repoRoot)

	gitPlainOpen = func(_ string) (*git.Repository, error) {
		return &git.Repository{}, nil
	}
	repoWorktree = func(_ *git.Repository) (*git.Worktree, error) {
		return nil, errors.New("worktree err")
	}
	_, err = loadGitDirStatus("/repo")
	assert.Error(t, err)

	repoWorktree = func(_ *git.Repository) (*git.Worktree, error) {
		return &git.Worktree{}, nil
	}
	worktreeStatus = func(_ *git.Worktree) (git.Status, error) {
		return nil, errors.New("status err")
	}
	_, err = loadGitDirStatus("/repo")
	assert.Error(t, err)

	worktreeStatus = func(_ *git.Worktree) (git.Status, error) {
		return git.Status{
			"dir/added.txt":    {Staging: git.Added},
			"dir/deleted.txt":  {Worktree: git.Deleted},
			"dir/modified.txt": {Worktree: git.Modified},
			"dir/clean.txt":    {Worktree: git.Unmodified, Staging: git.Unmodified},
			"other/file.txt":   {Worktree: git.Modified},
		}, nil
	}
	filepathRel = func(_, _ string) (string, error) {
		return "dir", nil
	}
	filepathFromSlashFn = func(s string) string {
		return s
	}
	result, err = loadGitDirStatus("/repo/dir")
	assert.NoError(t, err)
	assert.Len(t, result.entries, 3)
	assert.Equal(t, "added.txt", result.entries[0].displayName)

	filepathRel = func(_, _ string) (string, error) {
		return "", errors.New("rel err")
	}
	result, err = loadGitDirStatus("/repo/dir")
	assert.NoError(t, err)
	assert.NotEmpty(t, result.entries)
}

func TestLoadGitDirStatus_IgnoresMatcherEntries(t *testing.T) {
	t.Parallel()
	gitDirStatusTestLock.Lock()
	t.Cleanup(gitDirStatusTestLock.Unlock)
	origPlainOpen := gitPlainOpen
	origRepoWorktree := repoWorktree
	origStatus := worktreeStatus
	origRel := filepathRel
	origRoot := getRepositoryRoot
	origFromSlash := filepathFromSlashFn
	origLoadGlobalIgnore := loadGlobalIgnore
	origIsIgnoredPath := isIgnoredPath

	defer func() {
		gitPlainOpen = origPlainOpen
		repoWorktree = origRepoWorktree
		worktreeStatus = origStatus
		filepathRel = origRel
		getRepositoryRoot = origRoot
		filepathFromSlashFn = origFromSlash
		loadGlobalIgnore = origLoadGlobalIgnore
		isIgnoredPath = origIsIgnoredPath
	}()

	getRepositoryRoot = func(_ string) string {
		return "/repo"
	}
	gitPlainOpen = func(_ string) (*git.Repository, error) {
		return &git.Repository{}, nil
	}
	repoWorktree = func(_ *git.Repository) (*git.Worktree, error) {
		return &git.Worktree{}, nil
	}
	worktreeStatus = func(_ *git.Worktree) (git.Status, error) {
		return git.Status{
			"dir/ignored.txt": {Worktree: git.Modified},
		}, nil
	}
	filepathRel = func(_, _ string) (string, error) {
		return "dir", nil
	}
	filepathFromSlashFn = func(s string) string {
		return s
	}
	loadGlobalIgnore = func(_ string) gitignore.Matcher {
		return nil
	}
	isIgnoredPath = func(_ string, _ gitignore.Matcher) bool {
		return true
	}

	result, err := loadGitDirStatus("/repo/dir")
	assert.NoError(t, err)
	assert.Len(t, result.entries, 0)
}

func TestBadgeForStatus(t *testing.T) {
	t.Parallel()
	added := badgeForStatus(&git.FileStatus{Staging: git.Added})
	assert.Equal(t, "A", added.text)
	assert.Equal(t, "A:added", added.String())

	deleted := badgeForStatus(&git.FileStatus{Worktree: git.Deleted})
	assert.Equal(t, "D", deleted.text)

	modified := badgeForStatus(&git.FileStatus{Worktree: git.Modified})
	assert.Equal(t, "M", modified.text)

	renamed := badgeForStatus(&git.FileStatus{Worktree: git.Renamed})
	assert.Equal(t, "M", renamed.text)

	copied := badgeForStatus(&git.FileStatus{Worktree: git.Copied})
	assert.Equal(t, "M", copied.text)

	defaultBadge := badgeForStatus(&git.FileStatus{Worktree: git.Unmodified})
	assert.Equal(t, "?", defaultBadge.text)

	nilBadge := badgeForStatus(nil)
	assert.Equal(t, "?", nilBadge.text)
}

func TestGitDirStatusPreviewer_DefaultSeams(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	repo, err := git.PlainInit(tempDir, false)
	assert.NoError(t, err)

	worktree, err := repoWorktree(repo)
	assert.NoError(t, err)

	status, err := worktreeStatus(worktree)
	assert.NoError(t, err)
	assert.NotNil(t, status)
}
