package gitutils

import (
	"io"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

const maxGitDiffBytes = 1 * 1024 * 1024

func readLimitedContent(r io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxGitDiffBytes))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	if strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func countLines(content string) int {
	return len(splitLines(content))
}

func diffLineStats(oldContent, newContent string) (int, int) {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return 0, 0
	}
	matcher := difflib.NewMatcher(oldLines, newLines)
	insertions := 0
	deletions := 0
	for _, op := range matcher.GetOpCodes() {
		switch op.Tag {
		case 'r':
			deletions += op.I2 - op.I1
			insertions += op.J2 - op.J1
		case 'd':
			deletions += op.I2 - op.I1
		case 'i':
			insertions += op.J2 - op.J1
		}
	}
	return insertions, deletions
}
