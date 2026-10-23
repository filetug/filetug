package tui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/go-git/go-git/v5"
	"github.com/tuigoff/tuigoff/pkg/theme"
)

// gitScope says which list a git status belongs to.
type gitScope int

const (
	// gitTree is the status of the directories listed in the tree.
	gitTree gitScope = iota
	// gitFiles is the status of the entries of the file list.
	gitFiles
)

// gitWorkers is how many statuses are read at the same time.
const gitWorkers = 4

// gitReq asks for the status of one path.
type gitReq struct {
	Path  string
	IsDir bool
}

// gitStatusMsg carries the status text of one path. Results arrive one at a
// time: the receiver applies the message and, if it owns the stream, asks for
// the next with waitGit(msg.stream).
type gitStatusMsg struct {
	// Seq is the sequence of the request the status was read for.
	Seq   uint64
	Scope gitScope
	Path  string
	// Text is the styled status; never empty.
	Text   string
	stream <-chan gitStatusMsg
}

// Seams over git, replaced by tests.
var (
	gitRepoRoot = gitutils.GetRepositoryRoot
	gitOpen     = git.PlainOpen
	gitDirState = gitutils.GetDirStatus
	gitFileStat = gitutils.GetFileStatus
)

// gitStatusCmd reads the status of paths below dirPath off the event loop. A
// directory that is not in a repository, or no paths, end the stream at once.
func gitStatusCmd(ctx context.Context, seq uint64, scope gitScope, dirPath string, reqs []gitReq) tea.Cmd {
	return func() tea.Msg {
		repo := openRepo(dirPath)
		if repo == nil || len(reqs) == 0 {
			return nil
		}
		jobs := make(chan gitReq, len(reqs))
		for _, r := range reqs {
			jobs <- r
		}
		close(jobs)
		out := make(chan gitStatusMsg, len(reqs))
		var wg sync.WaitGroup
		for range min(gitWorkers, len(reqs)) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for r := range jobs {
					if ctx.Err() != nil {
						return
					}
					if text := gitLabel(ctx, repo, r); text != "" {
						out <- gitStatusMsg{Seq: seq, Scope: scope, Path: r.Path, Text: text}
					}
				}
			}()
		}
		go func() { wg.Wait(); close(out) }()
		return waitGit(out)()
	}
}

// waitGit waits for the next status of a stream; it returns nil when the
// stream has ended.
func waitGit(stream <-chan gitStatusMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-stream
		if !ok {
			return nil
		}
		msg.stream = stream
		return msg
	}
}

// openRepo opens the repository that contains dirPath, or returns nil.
func openRepo(dirPath string) *git.Repository {
	root := gitRepoRoot(dirPath)
	if root == "" {
		return nil
	}
	repo, err := gitOpen(root)
	if err != nil {
		return nil
	}
	return repo
}

// gitLabel reads the status of a path and formats it. A path with no changes
// has no label unless it is the root of the repository.
func gitLabel(ctx context.Context, repo *git.Repository, r gitReq) string {
	var status *gitutils.RepoStatus
	if r.IsDir {
		status = gitDirState(ctx, repo, r.Path)
	} else {
		status = gitFileStat(ctx, repo, r.Path)
	}
	if status == nil || ctx.Err() != nil {
		return ""
	}
	hasChanges := status.FilesChanged > 0 || status.Insertions > 0 || status.Deletions > 0
	if !hasChanges && (!r.IsDir || !isRepoRoot(r.Path)) {
		return ""
	}
	return formatGitStatus(status)
}

// isRepoRoot reports whether a directory is the top of its repository.
func isRepoRoot(dirPath string) bool {
	root := gitRepoRoot(dirPath)
	return root != "" && (dirPath == root || dirPath == root+"/")
}

// adaptive picks a colour by the theme variant.
func adaptive(dark, light string) color.Color {
	if theme.Dark {
		return lipgloss.Color(dark)
	}
	return lipgloss.Color(light)
}

// formatGitStatus renders a status as "┆branch┆ƒN+a-b": the branch, the number
// of changed files and the lines added and removed.
func formatGitStatus(s *gitutils.RepoStatus) string {
	muted := lipgloss.NewStyle().Foreground(theme.MutedColor())
	sep := muted.Render("┆")
	lines := lineCounts(s.Insertions, s.Deletions)
	if s.DirGitChangesStats == (gitutils.DirGitChangesStats{}) {
		return sep + muted.Render(s.Branch) + lines
	}
	return sep + muted.Render(s.Branch) + sep + muted.Render(fmt.Sprintf("ƒ%d", s.FilesChanged)) + lines
}

// lineCounts renders the lines added in green and removed in red, or ±0.
func lineCounts(added, removed int) string {
	var sb strings.Builder
	if added > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(adaptive("#3BD17F", "#1B7F46")).Render(fmt.Sprintf("+%d", added)))
	}
	if removed > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(theme.ErrorColor()).Render(fmt.Sprintf("-%d", removed)))
	}
	if sb.Len() == 0 {
		return lipgloss.NewStyle().Foreground(theme.MutedColor()).Render("±0")
	}
	return sb.String()
}
