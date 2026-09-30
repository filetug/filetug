package tui

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/tuigoff/tuigoff/pkg/nav/navtest"
	"github.com/tuigoff/tuigoff/pkg/uitest"
)

// tree builds a small directory tree and returns its root:
//
//	alpha/sub/
//	beta/
//	.hidden/
//	README.md
//	a.go
func tree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"alpha/sub", "beta", ".hidden", "Gamma"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(dir, "README.md"), "# Title\n\nhello\n")
	write(t, filepath.Join(dir, "a.go"), "package main\n\nfunc main() {}\n")
	return dir
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// open starts the application on dir under the test harness.
func open(t *testing.T, dir string) *navtest.Harness {
	t.Helper()
	isolateState(t)
	return navtest.New(t, rootPage(Options{Path: dir}), navtest.WithNav(shellOptions()...), navtest.WithSize(140, 30))
}

// openSized is open with a terminal of the given size.
func openSized(t *testing.T, dir string, w, h int) *navtest.Harness {
	t.Helper()
	isolateState(t)
	return navtest.New(t, rootPage(Options{Path: dir}), navtest.WithNav(shellOptions()...), navtest.WithSize(w, h))
}

// isolateState keeps the tests away from the user's saved state.
func isolateState(t *testing.T) {
	t.Helper()
	oldDir, oldTree, oldName := saveCurrentDir, saveSelectedTreeDir, saveCurrentFileName
	saveCurrentDir = func(string, string) {}
	saveSelectedTreeDir = func(string) {}
	saveCurrentFileName = func(string) {}
	t.Cleanup(func() { saveCurrentDir, saveSelectedTreeDir, saveCurrentFileName = oldDir, oldTree, oldName })
}

// fakeStore is a files.Store whose directories are given as entries.
type fakeStore struct {
	root    url.URL
	entries map[string][]os.DirEntry
	err     error
}

func (f fakeStore) RootTitle() string { return "fake/" }
func (f fakeStore) RootURL() url.URL  { return f.root }
func (f fakeStore) GetDirReader(context.Context, string) (files.DirReader, error) {
	return nil, files.ErrNotSupported
}
func (f fakeStore) ReadDir(_ context.Context, name string) ([]os.DirEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.entries[name], nil
}
func (f fakeStore) Delete(context.Context, string) error     { return nil }
func (f fakeStore) CreateDir(context.Context, string) error  { return nil }
func (f fakeStore) CreateFile(context.Context, string) error { return nil }

// osStore is the store of the real file system.
func osStore() files.Store { return osfile.NewStore("/") }

// stripANSI removes styling from rendered text.
func stripANSI(s string) string { return uitest.Plain(s) }
