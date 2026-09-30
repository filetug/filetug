package tui

import (
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/go-git/go-git/v5"
)

func sized(name string, size int64) os.DirEntry {
	return files.NewDirEntry(name, false, files.Size(size))
}

func TestSummarizeGroupsFilesByKindAndExtension(t *testing.T) {
	entries := []os.DirEntry{
		sized("b.go", 10), sized("a.go", 20), sized("pic.png", 100), sized("notes.txt", 5), sized("README", 1),
		sized("x.weird", 7), sized(".gitignore", 3), files.NewDirEntry("sub", true),
		sized("d.json", 4), sized("e.log", 2), failingEntry{files.NewDirEntry("bad.go", false)}, nilInfoEntry{files.NewDirEntry("nil.go", false)},
	}
	groups := summarize(entries)
	var titles []string
	for _, g := range groups {
		titles = append(titles, g.Title)
	}
	want := []string{"Code", "Data", "Images", "Logs", "Texts", "Others"}
	if len(titles) != len(want) {
		t.Fatalf("groups: %v", titles)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("groups are sorted, Others last: %v", titles)
		}
	}
	code := groups[0]
	if code.Count != 4 || code.Size != 30 || len(code.Exts) != 1 || code.Exts[0].Count != 4 {
		t.Fatalf("Go files: %+v %+v", code, code.Exts[0])
	}
	other := groups[5]
	if ids := other.Extensions(); len(ids) != 2 || ids[0] != "" || ids[1] != ".weird" {
		t.Fatalf("extensions of the group are sorted, the empty one first: %v", ids)
	}
	if len(summarize(nil)) != 0 {
		t.Fatal("no entries, no groups")
	}
}

func TestFileSize(t *testing.T) {
	if fileSize(sized("a", 9)) != 9 || fileSize(failingEntry{files.NewDirEntry("a", false)}) != 0 ||
		fileSize(nilInfoEntry{files.NewDirEntry("a", false)}) != 0 || fileSize(symlinkEntry{files.NewDirEntry("a", false)}) != 0 {
		t.Fatal("the size of a file, or 0 when unknown")
	}
}

func TestBuildSummary(t *testing.T) {
	if s, err := buildSummary(nil, "/x"); err != nil || s.Path != "/x" || s.Git != nil {
		t.Fatalf("no store: %+v %v", s, err)
	}
	if _, err := buildSummary(fakeStore{err: errors.New("boom")}, "/x"); err == nil {
		t.Fatal("a read error")
	}
	remote := fakeStore{root: url.URL{Scheme: "ftp"}, entries: map[string][]os.DirEntry{"/x": {sized("a.go", 1)}}}
	if s, err := buildSummary(remote, "/x"); err != nil || len(s.Groups) != 1 || s.Git != nil {
		t.Fatalf("a remote directory has no git: %+v %v", s, err)
	}
	fakeGit(t, "/repo", nil)
	gitOpen = func(string) (*git.Repository, error) { return nil, errors.New("corrupt") }
	local := fakeStore{root: url.URL{Scheme: "file"}, entries: map[string][]os.DirEntry{"/repo": nil}}
	if s, err := buildSummary(local, "/repo"); err != nil || s.Git == nil || s.Git.Err == nil {
		t.Fatalf("a local directory in a repository has a git status: %+v %v", s, err)
	}
}
