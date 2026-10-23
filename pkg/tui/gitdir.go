package tui

import (
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
)

// Seams over git, replaced by tests.
var (
	gitWorktreeOf   = func(repo *git.Repository) (*git.Worktree, error) { return repo.Worktree() }
	gitWorktreeStat = func(worktree *git.Worktree) (git.Status, error) { return worktree.Status() }
	filepathRel     = filepath.Rel
	loadGitIgnore   = gitutils.LoadGlobalIgnoreMatcher
	isIgnoredInGit  = gitutils.IsIgnoredPath
	stageInGit      = gitutils.StageFile
	unstageInGit    = gitutils.UnstageFile
)

// gitBadge is the letter and meaning of a file's change.
type gitBadge struct {
	Text  string
	Label string
}

// gitEntry is one changed file of a directory.
type gitEntry struct {
	FullPath    string
	DisplayName string
	Staged      bool
	Badge       gitBadge
}

// gitDirStatus is the changed files of the directory, as the repository sees
// them.
type gitDirStatus struct {
	RepoRoot string
	Entries  []gitEntry
	Err      error
}

// loadGitDirStatus reads the changed files of a directory. A directory outside a
// repository returns nil.
func loadGitDirStatus(dirPath string) *gitDirStatus {
	repoRoot := gitRepoRoot(dirPath)
	if repoRoot == "" {
		return nil
	}
	result := &gitDirStatus{RepoRoot: repoRoot}
	repo, err := gitOpen(repoRoot)
	if err != nil {
		result.Err = err
		return result
	}
	worktree, err := gitWorktreeOf(repo)
	if err != nil {
		result.Err = err
		return result
	}
	status, err := gitWorktreeStat(worktree)
	if err != nil {
		result.Err = err
		return result
	}
	matcher := loadGitIgnore(repoRoot)
	prefix := ""
	if rel, err := filepathRel(repoRoot, dirPath); err == nil && rel != "." {
		prefix = filepath.ToSlash(rel) + "/"
	}
	for fileName, fileStatus := range status {
		slash := filepath.ToSlash(fileName)
		if (prefix != "" && !strings.HasPrefix(slash, prefix)) ||
			(fileStatus.Worktree == git.Unmodified && fileStatus.Staging == git.Unmodified) ||
			isIgnoredInGit(slash, matcher) {
			continue
		}
		result.Entries = append(result.Entries, gitEntry{
			FullPath:    filepath.Join(repoRoot, filepath.FromSlash(slash)),
			DisplayName: strings.TrimPrefix(slash, prefix),
			Staged:      fileStatus.Staging != git.Unmodified,
			Badge:       badgeForStatus(fileStatus),
		})
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].DisplayName < result.Entries[j].DisplayName })
	return result
}

// badgeForStatus names the change a file status stands for.
func badgeForStatus(status *git.FileStatus) gitBadge {
	switch {
	case status == nil:
		return gitBadge{Text: "?", Label: "changed"}
	case status.Staging == git.Added || status.Worktree == git.Untracked:
		return gitBadge{Text: "A", Label: "added"}
	case status.Staging == git.Deleted || status.Worktree == git.Deleted:
		return gitBadge{Text: "D", Label: "deleted"}
	case status.Staging == git.Modified || status.Worktree == git.Modified,
		status.Staging == git.Renamed || status.Worktree == git.Renamed,
		status.Staging == git.Copied || status.Worktree == git.Copied:
		return gitBadge{Text: "M", Label: "changed"}
	}
	return gitBadge{Text: "?", Label: "changed"}
}

// gitDirLoadedMsg is a fresh git status of a directory.
type gitDirLoadedMsg struct {
	Path   string
	Status *gitDirStatus
}

// gitDirCmd reads the git status of a directory again, after a file was staged
// or unstaged.
func gitDirCmd(dirPath string) tea.Cmd {
	return func() tea.Msg { return gitDirLoadedMsg{Path: dirPath, Status: loadGitDirStatus(dirPath)} }
}

// stageCmd stages or unstages a file off the event loop and then reads the
// status of the directory again.
func stageCmd(dirPath string, entry gitEntry) tea.Cmd {
	return func() tea.Msg {
		var err error
		if entry.Staged {
			err = unstageInGit(entry.FullPath)
		} else {
			err = stageInGit(entry.FullPath)
		}
		status := loadGitDirStatus(dirPath)
		if err != nil {
			if status == nil {
				status = &gitDirStatus{}
			}
			status.Err = err
		}
		return gitDirLoadedMsg{Path: dirPath, Status: status}
	}
}
