package filetug

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBasket_AddToBasket(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}

	var b Basket
	b.AddToBasket(entries[0])
	if got := len(b.entries); got != 1 {
		t.Fatalf("expected 1 entry after first add, got %d", got)
	}

	b.AddToBasket(entries[1])
	if got := len(b.entries); got != 2 {
		t.Fatalf("expected 2 entries after second add, got %d", got)
	}
	if b.entries[0] != entries[0] || b.entries[1] != entries[1] {
		t.Error("expected entries to be appended in order")
	}
}

func TestBasket_Clear(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 entry")
	}

	b := Basket{entries: []os.DirEntry{entries[0]}}
	b.Clear()

	if got := len(b.entries); got != 0 {
		t.Fatalf("expected empty basket after Clear, got %d", got)
	}
}
