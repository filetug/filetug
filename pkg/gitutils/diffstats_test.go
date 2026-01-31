package gitutils

import (
	"errors"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read boom")
}

func TestReadLimitedContent(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		content, err := readLimitedContent(strings.NewReader("alpha\nbeta"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if content != "alpha\nbeta" {
			t.Fatalf("expected content to match, got %q", content)
		}
	})

	t.Run("error", func(t *testing.T) {
		_, err := readLimitedContent(errReader{})
		if err == nil {
			t.Fatal("expected error from readLimitedContent")
		}
	})
}

func TestSplitLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{name: "empty", input: "", wantLen: 0},
		{name: "single_no_newline", input: "line", wantLen: 1},
		{name: "single_with_newline", input: "line\n", wantLen: 1},
		{name: "multi", input: "a\nb\nc", wantLen: 3},
		{name: "multi_with_trailing", input: "a\nb\n", wantLen: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.input)
			if len(lines) != tt.wantLen {
				t.Fatalf("expected %d lines, got %d", tt.wantLen, len(lines))
			}
		})
	}
}

func TestDiffLineStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		oldContent string
		newContent string
		insertions int
		deletions  int
	}{
		{name: "empty", oldContent: "", newContent: "", insertions: 0, deletions: 0},
		{name: "replace", oldContent: "a\nb\n", newContent: "a\nc\n", insertions: 1, deletions: 1},
		{name: "delete", oldContent: "a\nb\n", newContent: "a\n", insertions: 0, deletions: 1},
		{name: "insert", oldContent: "a\n", newContent: "a\nb\n", insertions: 1, deletions: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins, del := diffLineStats(tt.oldContent, tt.newContent)
			if ins != tt.insertions || del != tt.deletions {
				t.Fatalf("expected +%d -%d, got +%d -%d", tt.insertions, tt.deletions, ins, del)
			}
		})
	}
}
