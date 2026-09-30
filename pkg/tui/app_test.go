package tui

import (
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/tuigoff/tuigoff/pkg/nav"
)

// stubEnv replaces the environment seams and restores them afterwards.
func stubEnv(t *testing.T) {
	t.Helper()
	oldState, oldAbs, oldStat, oldOS, oldHTTP, oldFTP := getState, absPath, statPath, newOSFile, newHTTP, newFTP
	t.Cleanup(func() {
		getState, absPath, statPath, newOSFile, newHTTP, newFTP = oldState, oldAbs, oldStat, oldOS, oldHTTP, oldFTP
	})
	newOSFile = func() files.Store { return fakeStore{root: url.URL{Scheme: "file"}} }
	newHTTP = func(u url.URL) files.Store { return fakeStore{root: u} }
	newFTP = func(u url.URL) files.Store { return fakeStore{root: u} }
}

func TestStartForAPath(t *testing.T) {
	stubEnv(t)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "f.txt"), "x")

	if st := startFor(Options{Path: dir}); st.dir != dir || st.entry != "" {
		t.Fatalf("a directory: %+v", st)
	}
	if st := startFor(Options{Path: filepath.Join(dir, "f.txt")}); st.dir != dir || st.entry != "f.txt" {
		t.Fatalf("a file opens its directory and selects it: %+v", st)
	}
	if st := startFor(Options{Path: filepath.Join(dir, "missing")}); st.dir != filepath.Join(dir, "missing") || st.entry != "" {
		t.Fatalf("a missing path is opened as it is: %+v", st)
	}
	absPath = func(string) (string, error) { return "", errors.New("no cwd") }
	if st := startFor(Options{Path: "rel"}); st.dir != "rel" {
		t.Fatalf("an unresolvable path is kept: %+v", st)
	}
}

func TestPersistedStart(t *testing.T) {
	stubEnv(t)
	state := func(s *ftstate.State, err error) { getState = func() (*ftstate.State, error) { return s, err } }

	state(nil, errors.New("no state"))
	if st := startFor(Options{}); st.dir != "~" || st.entry != "" {
		t.Fatalf("defaults: %+v", st)
	}
	state(&ftstate.State{}, nil)
	if st := startFor(Options{}); st.dir != "~" {
		t.Fatalf("an empty state opens home: %+v", st)
	}
	state(&ftstate.State{Store: "file:", CurrentDir: "/work", CurrentDirEntry: "x.go"}, nil)
	if st := startFor(Options{}); st.dir != "/work" || st.entry != "x.go" {
		t.Fatalf("a saved location: %+v", st)
	}
	state(&ftstate.State{CurrentDir: "/work", CurrentDirEntry: "x.go"}, errors.New("read"))
	if st := startFor(Options{}); st.entry != "" {
		t.Fatalf("a state read with an error does not select a file: %+v", st)
	}
	state(&ftstate.State{Store: "https://example.com/files"}, nil)
	if st := startFor(Options{}); st.store.RootURL().Host != "example.com" {
		t.Fatalf("an HTTP store: %+v", st.store.RootURL())
	}
	state(&ftstate.State{Store: "ftp://example.org/pub"}, nil)
	if st := startFor(Options{}); st.store.RootURL().Scheme != "ftp" {
		t.Fatalf("an FTP store: %+v", st.store.RootURL())
	}
	newFTP = func(url.URL) files.Store { return nil }
	if st := startFor(Options{}); st.store.RootURL().Scheme != "file" {
		t.Fatalf("an FTP store that cannot be created falls back: %+v", st.store.RootURL())
	}
	state(&ftstate.State{Store: "ftp://%zz"}, nil)
	if st := startFor(Options{}); st.store.RootURL().Scheme != "file" {
		t.Fatalf("an invalid store URL falls back: %+v", st.store.RootURL())
	}
	state(&ftstate.State{CurrentDir: "https://example.com/a/b"}, nil)
	if st := startFor(Options{}); st.dir != "/a/b" || st.store.RootURL().Path != "/" || st.store.RootURL().Host != "example.com" {
		t.Fatalf("an HTTP location carries its store: %+v %v", st, st.store.RootURL())
	}
	state(&ftstate.State{CurrentDir: "https://exa mple.com/%zz"}, nil)
	if st := startFor(Options{}); st.dir != "https://exa mple.com/%zz" {
		t.Fatalf("an invalid location is kept: %+v", st)
	}
}

func TestDefaultEnvironmentSeams(t *testing.T) {
	if newOSFile().RootURL().Scheme != "file" {
		t.Fatal("the local store")
	}
	if newHTTP(url.URL{Scheme: "https", Host: "example.com"}).RootURL().Host != "example.com" {
		t.Fatal("the HTTP store")
	}
	if s := newFTP(url.URL{Scheme: "ftp", Host: "127.0.0.1:1"}); s != nil && s.RootURL().Scheme != "ftp" {
		t.Fatal("the FTP store")
	}
}

func TestActionsAreBoundToAltAndOption(t *testing.T) {
	keys := map[string][]string{}
	for _, a := range actions() {
		keys[a.ID] = a.Binding.Keys()
	}
	for id, want := range map[string][]string{
		"root": {"alt+/", "÷", "alt+r", "®"},
		"home": {"alt+~", "alt+`", "alt+h", "˙"},
		"exit": {"alt+x", "≈"},
	} {
		got := append([]string(nil), keys[id]...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("%s: %v, want %v", id, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: %v, want %v", id, got, want)
			}
		}
	}
}

func TestAltShortcutsNavigate(t *testing.T) {
	dir := tree(t)
	h := open(t, dir)
	h.Press("alt+/")
	if browserOf(h).sess.treeRoot.Path() != "/" {
		t.Fatal("Alt+/ goes to the root")
	}
	h.Press("®") // Option+r on a Mac keyboard
	h.Press("alt+~")
	home, _ := os.UserHomeDir()
	if browserOf(h).sess.treeRoot.Path() != home {
		t.Fatal("Alt+~ goes home")
	}
	h.Press("alt+x")
	if !h.Quit() {
		t.Fatal("Alt+X exits")
	}
}

func TestNewAndRun(t *testing.T) {
	stubEnv(t)
	isolateState(t)
	getState = func() (*ftstate.State, error) { return nil, errors.New("none") }
	var ran nav.Model
	old := runShell
	runShell = func(m nav.Model, _ ...tea.ProgramOption) error { ran = m; return errors.New("ended") }
	defer func() { runShell = old }()

	if err := Run(Options{}); err == nil || err.Error() != "ended" {
		t.Fatalf("Run returns what the program returns: %v", err)
	}
	if ran.Depth() != 1 || ran.Breadcrumbs()[0].Title != "FileTug" {
		t.Fatal("Run runs the application")
	}
	if New(Options{}).Zone() != nav.FocusToMenu {
		t.Fatal("the tree starts focused")
	}
}

func TestRunShellRunsTheProgram(t *testing.T) {
	stubEnv(t)
	isolateState(t)
	getState = func() (*ftstate.State, error) { return nil, errors.New("none") }
	// Ctrl+Q arrives on the program's input, so no terminal is needed.
	err := Run(Options{}, tea.WithInput(strings.NewReader("\x11")), tea.WithOutput(io.Discard))
	if err != nil {
		t.Fatalf("the program ends when it is told to quit: %v", err)
	}
}

func TestNewFTPRejectsOtherSchemes(t *testing.T) {
	if s := newFTP(url.URL{Scheme: "http"}); s != nil {
		t.Fatal("not an FTP address")
	}
}
