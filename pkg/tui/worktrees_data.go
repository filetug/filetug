package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type worktreeInfo struct {
	Path       string
	Head       string
	Branch     string
	Canonical  bool
	Detached   bool
	Locked     bool
	Prunable   bool
	Dirty      bool
	StatusRead bool
	LastCommit time.Time
	WB         *wbWorktreeInfo
}

type wbWorktreeInfo struct {
	Path         string    `json:"path"`
	CanonicalDir string    `json:"canonical_dir"`
	Repository   string    `json:"repository"`
	Layout       string    `json:"layout"`
	Branch       string    `json:"branch"`
	EffortID     string    `json:"effort_id"`
	ParentEffort string    `json:"parent_effort,omitempty"`
	RootEffort   string    `json:"root_effort"`
	HasManifest  bool      `json:"has_manifest"`
	Provenance   string    `json:"provenance,omitempty"`
	LastCommit   time.Time `json:"last_commit,omitempty"`
	Dirty        bool      `json:"dirty"`
	Missing      bool      `json:"missing"`
	Merged       bool      `json:"merged_into_target"`
	OwnerState   string    `json:"owner_state"`
	OwnerAgent   string    `json:"owner_agent,omitempty"`
	OwnerPID     int       `json:"owner_pid,omitempty"`
	Disposition  string    `json:"disposition"`
	Evidence     []string  `json:"evidence"`
}

type wbWorktreeFamily struct {
	Worktrees []wbWorktreeInfo `json:"worktrees"`
}

type wbOrphanReport struct {
	Families []wbWorktreeFamily `json:"families"`
}

var worktreeExecCommandContext = exec.CommandContext
var worktreeExecCommand = exec.Command
var worktreeExecLookPath = exec.LookPath

func readGitWorktrees(ctx context.Context, repoRoot string) (string, []worktreeInfo, error) {
	canonical, err := canonicalWorktreePath(ctx, repoRoot)
	if err != nil {
		return "", nil, err
	}
	out, err := worktreeExecCommandContext(ctx, "git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", nil, fmt.Errorf("git worktree list: %w", err)
	}
	items, err := parseGitWorktrees(string(out), canonical)
	if err != nil {
		return canonical, nil, err
	}
	return canonical, items, nil
}

func parseGitWorktrees(output, canonical string) ([]worktreeInfo, error) {
	items := make([]worktreeInfo, 0)
	var item worktreeInfo
	flush := func() {
		if item.Path == "" {
			return
		}
		item.Path = filepath.Clean(item.Path)
		item.Canonical = item.Path == filepath.Clean(canonical)
		items = append(items, item)
		item = worktreeInfo{}
	}
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			item.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			item.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			item.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "detached":
			item.Detached = true
		case line == "locked" || strings.HasPrefix(line, "locked "):
			item.Locked = true
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			item.Prunable = true
		case line == "":
			flush()
		}
	}
	flush()
	if len(items) == 0 {
		return nil, errors.New("git reported no worktrees")
	}
	sortWorktrees(items)
	return items, nil
}

func canonicalWorktreePath(ctx context.Context, repoRoot string) (string, error) {
	out, err := worktreeExecCommandContext(ctx, "git", "-C", repoRoot, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git common directory: %w", err)
	}
	common := filepath.Clean(strings.TrimSpace(string(out)))
	if filepath.Base(common) != ".git" {
		return "", fmt.Errorf("unsupported Git common directory %s", common)
	}
	return filepath.Dir(common), nil
}

func gitRepositoryRoot(path string) string {
	out, err := worktreeExecCommand("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(string(out)))
}

func enrichGitWorktrees(ctx context.Context, items []worktreeInfo) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	for index := range items {
		if items[index].Prunable {
			continue
		}
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			status, err := worktreeExecCommandContext(ctx, "git", "-C", items[index].Path, "status", "--porcelain").Output()
			if err == nil {
				items[index].Dirty = len(status) > 0
				items[index].StatusRead = true
			}
			commit, err := worktreeExecCommandContext(ctx, "git", "-C", items[index].Path, "log", "-1", "--format=%cI").Output()
			if err == nil {
				items[index].LastCommit, _ = time.Parse(time.RFC3339, strings.TrimSpace(string(commit)))
			}
		}(index)
	}
	wg.Wait()
}

func readWBWorktreeMetadata(ctx context.Context, canonical string) (map[string]wbWorktreeInfo, string) {
	metadata := make(map[string]wbWorktreeInfo)
	wb, err := worktreeExecLookPath("wb")
	if err != nil {
		return metadata, "WB not installed; showing raw Git"
	}
	projectsRoot := filepath.Dir(filepath.Dir(canonical))
	command := worktreeExecCommandContext(ctx, wb, "worktree", "orphans", "--format", "json", "--non-interactive", "--projects-root", projectsRoot)
	out, err := command.Output()
	if err != nil {
		return metadata, "WB metadata unavailable"
	}
	var report wbOrphanReport
	if err := json.Unmarshal(out, &report); err != nil {
		return metadata, "WB returned unsupported JSON"
	}
	for _, family := range report.Families {
		for _, worktree := range family.Worktrees {
			if filepath.Clean(worktree.CanonicalDir) == filepath.Clean(canonical) {
				metadata[filepath.Clean(worktree.Path)] = worktree
			}
		}
	}
	return metadata, "WB metadata loaded"
}

func repositoryLabel(repoRoot string) string {
	clean := filepath.Clean(repoRoot)
	return filepath.Base(filepath.Dir(clean)) + "/" + filepath.Base(clean)
}

func shortSHA(value string) string {
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func sortWorktrees(items []worktreeInfo) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Canonical != items[j].Canonical {
			return items[i].Canonical
		}
		return items[i].Branch < items[j].Branch
	})
}
