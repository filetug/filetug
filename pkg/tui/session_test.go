package tui

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/filetug/filetug/pkg/files"
)

func TestSessionBeginCancelsTheRequestInFlight(t *testing.T) {
	s := &session{}
	ctx1, seq1 := s.begin()
	ctx2, seq2 := s.begin()
	if ctx1.Err() == nil {
		t.Fatal("the first request must be canceled by the second")
	}
	if ctx2.Err() != nil || seq2 != seq1+1 {
		t.Fatalf("the second request is current: err=%v seq=%d,%d", ctx2.Err(), seq1, seq2)
	}
}

func TestSessionCurrentPath(t *testing.T) {
	s := &session{}
	if s.currentPath() != "" {
		t.Fatal("no directory yet")
	}
	s.current = files.NewDirContext(nil, "/a", nil)
	if s.currentPath() != "/a" {
		t.Fatal("the path of the current directory")
	}
}

func TestReadDir(t *testing.T) {
	if _, err := readDir(context.Background(), nil, "/"); !errors.Is(err, errNoStore) {
		t.Fatalf("no store: %v", err)
	}
	boom := errors.New("boom")
	if _, err := readDir(context.Background(), fakeStore{err: boom}, "/"); !errors.Is(err, boom) {
		t.Fatalf("read error: %v", err)
	}
	store := fakeStore{entries: map[string][]os.DirEntry{"/": {
		files.NewDirEntry("b.txt", false), files.NewDirEntry("z", true),
		files.NewDirEntry("a.txt", false), files.NewDirEntry("m", true),
	}}}
	dir, err := readDir(context.Background(), store, "/")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range dir.Children() {
		names = append(names, c.Name())
	}
	want := []string{"m", "z", "a.txt", "b.txt"}
	if len(names) != 4 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] || names[3] != want[3] {
		t.Fatalf("directories first, then by name: %v", names)
	}
}

func TestLoadDirCmdReportsTheSequence(t *testing.T) {
	store := fakeStore{root: url.URL{Scheme: "fake"}, entries: map[string][]os.DirEntry{"/": nil}}
	msg := loadDirCmd(context.Background(), store, "/", 7, true)().(dirLoadedMsg)
	if msg.Seq != 7 || !msg.Root || msg.Path != "/" || msg.Err != nil || msg.Dir == nil {
		t.Fatalf("unexpected result: %+v", msg)
	}
}
