package gitutils

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

var (
	execCommand   = exec.Command
	osReadFile    = os.ReadFile
	osUserHomeDir = os.UserHomeDir
)

// LoadGlobalIgnoreMatcher loads ignore patterns from the configured global excludes file.
// If git config is unavailable or empty, it falls back to ignoring .DS_Store.
func LoadGlobalIgnoreMatcher(repoRoot string) gitignore.Matcher {
	patterns := loadGlobalIgnorePatterns(repoRoot)
	if len(patterns) == 0 {
		return nil
	}
	matcher := gitignore.NewMatcher(patterns)
	return matcher
}

// IsIgnoredPath returns true if a path should be ignored by the matcher.
func IsIgnoredPath(path string, matcher gitignore.Matcher) bool {
	if matcher == nil {
		return false
	}
	segments := strings.Split(path, "/")
	ignored := matcher.Match(segments, false)
	return ignored
}

func loadGlobalIgnorePatterns(repoRoot string) []gitignore.Pattern {
	patterns := make([]gitignore.Pattern, 0)
	excludesPath, ok := getGlobalExcludesFile(repoRoot)
	if ok {
		filePatterns, err := loadIgnorePatternsFromFile(excludesPath)
		if err == nil {
			patterns = append(patterns, filePatterns...)
		}
		return patterns
	}
	defaultPattern := gitignore.ParsePattern(".DS_Store", nil)
	patterns = append(patterns, defaultPattern)
	return patterns
}

func loadIgnorePatternsFromFile(path string) ([]gitignore.Pattern, error) {
	content, err := osReadFile(path)
	if err != nil {
		return nil, err
	}
	patterns := parseIgnorePatterns(content)
	return patterns, nil
}

func parseIgnorePatterns(content []byte) []gitignore.Pattern {
	patterns := make([]gitignore.Pattern, 0)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "#") {
			continue
		}
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		pattern := gitignore.ParsePattern(line, nil)
		patterns = append(patterns, pattern)
	}
	return patterns
}

func getGlobalExcludesFile(repoRoot string) (string, bool) {
	cmd := execCommand("git", "-C", repoRoot, "config", "--get", "core.excludesFile")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return "", false
	}
	if raw == "~" || strings.HasPrefix(raw, "~/") {
		home, homeErr := osUserHomeDir()
		if homeErr == nil {
			if raw == "~" {
				raw = home
			} else {
				suffix := strings.TrimPrefix(raw, "~/")
				raw = filepath.Join(home, suffix)
			}
		}
	}
	return raw, true
}
