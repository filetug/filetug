package viewers

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/gitutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/gdamore/tcell/v2"
	"github.com/go-git/go-git/v5"
	"github.com/rivo/tview"
)

var (
	gitPlainOpen        = git.PlainOpen
	repoWorktree        = func(repo *git.Repository) (*git.Worktree, error) { return repo.Worktree() }
	worktreeStatus      = func(worktree *git.Worktree) (git.Status, error) { return worktree.Status() }
	filepathRel         = filepath.Rel
	getRepositoryRoot   = gitutils.GetRepositoryRoot
	filepathFromSlashFn = filepath.FromSlash
	loadGlobalIgnore    = gitutils.LoadGlobalIgnoreMatcher
	isIgnoredPath       = gitutils.IsIgnoredPath
)

type GitDirStatusPreviewer struct {
	*sneatv.Boxed
	table *tview.Table

	dir string

	entries []gitDirStatusEntry

	queueUpdateDraw func(func())
	statusLoader    func(string) (gitDirStatusResult, error)
	stageFile       func(string) error
	unstageFile     func(string) error
}

type gitDirStatusEntry struct {
	fullPath    string
	displayName string
	staged      bool
	badge       gitBadge
}

type gitDirStatusResult struct {
	repoRoot string
	entries  []gitDirStatusEntry
}

type gitBadge struct {
	text  string
	color tcell.Color
	label string
}

func NewGitDirStatusPreviewer() *GitDirStatusPreviewer {
	table := tview.NewTable()
	table.SetSelectable(true, false)

	p := &GitDirStatusPreviewer{
		table:        table,
		statusLoader: loadGitDirStatus,
		stageFile:    gitutils.StageFile,
		unstageFile:  gitutils.UnstageFile,
	}
	p.Boxed = sneatv.NewBoxed(table)

	selectedStyle := tcell.StyleDefault
	selectedStyle = selectedStyle.Foreground(tcell.ColorBlack)
	selectedStyle = selectedStyle.Background(tcell.ColorWhiteSmoke)
	p.table.SetSelectedStyle(selectedStyle)
	p.table.SetInputCapture(p.handleInput)

	return p
}

func (p *GitDirStatusPreviewer) Preview(entry files.EntryWithDirPath, _ []byte, queueUpdateDraw func(func())) {
	dirPath := entry.Dir
	if entry.IsDir() {
		dirPath = entry.FullName()
	}
	p.SetDir(dirPath, queueUpdateDraw)
}

func (p *GitDirStatusPreviewer) Main() tview.Primitive {
	return p
}

func (p *GitDirStatusPreviewer) Meta() tview.Primitive {
	return nil
}

func (p *GitDirStatusPreviewer) SetDir(dirPath string, queueUpdateDraw func(func())) {
	p.dir = dirPath
	p.queueUpdateDraw = queueUpdateDraw
	p.setMessage("Loading...", tcell.ColorLightGray)
	go p.refresh()
}

func (p *GitDirStatusPreviewer) handleInput(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
		row, _ := p.table.GetSelection()
		if row < 0 || row >= len(p.entries) {
			return nil
		}
		entry := p.entries[row]
		if entry.staged {
			err := p.unstageFile(entry.fullPath)
			if err != nil {
				p.setError(err)
				return nil
			}
		} else {
			err := p.stageFile(entry.fullPath)
			if err != nil {
				p.setError(err)
				return nil
			}
		}
		go p.refresh()
		return nil
	}
	return event
}

func (p *GitDirStatusPreviewer) refresh() {
	result, err := p.statusLoader(p.dir)
	if err != nil {
		p.queueUpdate(func() {
			p.setError(err)
		})
		return
	}
	p.queueUpdate(func() {
		if result.repoRoot == "" {
			p.entries = nil
			p.setMessage("Not a git repository", tcell.ColorGray)
			return
		}
		if len(result.entries) == 0 {
			p.entries = nil
			p.setMessage("No changes", tcell.ColorGray)
			return
		}
		p.entries = result.entries
		p.renderEntries()
	})
}

func (p *GitDirStatusPreviewer) renderEntries() {
	p.table.Clear()
	for row, entry := range p.entries {
		checkbox := " "
		if entry.staged {
			checkbox = "✓"
		}
		nameText := checkbox + " " + entry.displayName
		nameCell := tview.NewTableCell(nameText)
		p.table.SetCell(row, 0, nameCell)

		badgeCell := tview.NewTableCell(entry.badge.text)
		badgeCell.SetTextColor(entry.badge.color)
		badgeCell.SetAlign(tview.AlignCenter)
		badgeCell.SetExpansion(1)
		p.table.SetCell(row, 1, badgeCell)
	}
	p.table.SetSelectable(true, false)
}

func (p *GitDirStatusPreviewer) setMessage(text string, color tcell.Color) {
	p.table.Clear()
	cell := tview.NewTableCell(text)
	cell.SetTextColor(color)
	p.table.SetCell(0, 0, cell)
	p.table.SetSelectable(false, false)
}

func (p *GitDirStatusPreviewer) setError(err error) {
	text := err.Error()
	p.setMessage(text, tcell.ColorOrangeRed)
}

func (p *GitDirStatusPreviewer) queueUpdate(f func()) {
	if p.queueUpdateDraw != nil {
		p.queueUpdateDraw(f)
		return
	}
	f()
}

func loadGitDirStatus(dirPath string) (gitDirStatusResult, error) {
	repoRoot := getRepositoryRoot(dirPath)
	if repoRoot == "" {
		return gitDirStatusResult{}, nil
	}
	repo, err := gitPlainOpen(repoRoot)
	if err != nil {
		return gitDirStatusResult{repoRoot: repoRoot}, err
	}
	worktree, err := repoWorktree(repo)
	if err != nil {
		return gitDirStatusResult{repoRoot: repoRoot}, err
	}
	status, err := worktreeStatus(worktree)
	if err != nil {
		return gitDirStatusResult{repoRoot: repoRoot}, err
	}

	matcher := loadGlobalIgnore(repoRoot)

	relDir, err := filepathRel(repoRoot, dirPath)
	if err != nil || relDir == "." {
		relDir = ""
	}
	relDir = filepath.ToSlash(relDir)

	prefix := ""
	if relDir != "" {
		prefix = relDir + "/"
	}

	entries := make([]gitDirStatusEntry, 0, len(status))
	for fileName, fileStatus := range status {
		fileNameSlash := filepath.ToSlash(fileName)
		if prefix != "" && !strings.HasPrefix(fileNameSlash, prefix) {
			continue
		}
		if fileStatus.Worktree == git.Unmodified && fileStatus.Staging == git.Unmodified {
			continue
		}
		if isIgnoredPath(fileNameSlash, matcher) {
			continue
		}

		displayName := fileNameSlash
		if prefix != "" {
			displayName = strings.TrimPrefix(fileNameSlash, prefix)
		}
		absPath := filepath.Join(repoRoot, filepathFromSlashFn(fileNameSlash))
		entry := gitDirStatusEntry{
			fullPath:    absPath,
			displayName: displayName,
			staged:      fileStatus.Staging != git.Unmodified,
			badge:       badgeForStatus(fileStatus),
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].displayName < entries[j].displayName
	})

	return gitDirStatusResult{
		repoRoot: repoRoot,
		entries:  entries,
	}, nil
}

func badgeForStatus(status *git.FileStatus) gitBadge {
	if status == nil {
		return gitBadge{text: "?", color: tcell.ColorGray, label: "changed"}
	}
	if status.Staging == git.Added || status.Worktree == git.Untracked {
		return gitBadge{text: "A", color: tcell.ColorLightGreen, label: "added"}
	}
	if status.Staging == git.Deleted || status.Worktree == git.Deleted {
		return gitBadge{text: "D", color: tcell.ColorOrangeRed, label: "deleted"}
	}
	if status.Staging == git.Modified || status.Worktree == git.Modified {
		return gitBadge{text: "M", color: tcell.ColorYellow, label: "changed"}
	}
	if status.Staging == git.Renamed || status.Worktree == git.Renamed {
		return gitBadge{text: "M", color: tcell.ColorYellow, label: "changed"}
	}
	if status.Staging == git.Copied || status.Worktree == git.Copied {
		return gitBadge{text: "M", color: tcell.ColorYellow, label: "changed"}
	}
	return gitBadge{text: "?", color: tcell.ColorGray, label: "changed"}
}

func (b gitBadge) String() string {
	return fmt.Sprintf("%s:%s", b.text, b.label)
}
