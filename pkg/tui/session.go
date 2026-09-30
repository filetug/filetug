package tui

import (
	"context"
	"errors"
	"os"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
)

// session is the state shared by the screens of the shell. It is only touched
// from Update, never from a tea.Cmd: commands capture the values they need and
// report back with messages that carry the sequence number they were started
// with.
type session struct {
	store files.Store
	// seq numbers the requests that replace the directory being shown; a result
	// whose sequence is not the current one is stale and is dropped.
	seq    uint64
	cancel context.CancelFunc
	// treeRoot is the directory the tree lists the sub-directories of.
	treeRoot *files.DirContext
	// current is the directory whose entries the file list shows: the tree's
	// root, or the sub-directory the tree cursor is on.
	current *files.DirContext
	// entryName is the file name that is selected again when a directory is
	// shown.
	entryName string
	// previewSeq numbers preview requests in the same way seq does for
	// directories.
	previewSeq uint64
}

// begin cancels the request in flight and starts a new one.
func (s *session) begin() (ctx context.Context, seq uint64) {
	if s.cancel != nil {
		s.cancel()
	}
	ctx, s.cancel = context.WithCancel(context.Background())
	s.seq++
	return ctx, s.seq
}

// currentPath is the path of the directory being shown, or "".
func (s *session) currentPath() string {
	if s.current == nil {
		return ""
	}
	return s.current.Path()
}

// goDirMsg makes a directory the root of the tree and shows it. A nil Store
// keeps the current store.
type goDirMsg struct {
	Store files.Store
	Path  string
}

// showDirMsg shows a directory in the file list without changing the root of
// the tree; the tree sends it when its cursor moves to a sub-directory.
type showDirMsg struct{ Path string }

// dirLoadedMsg is the result of reading a directory.
type dirLoadedMsg struct {
	Seq uint64
	// Root is true when the directory became the root of the tree.
	Root bool
	Path string
	Dir  *files.DirContext
	Err  error
}

// loadDirCmd reads a directory off the event loop.
func loadDirCmd(ctx context.Context, store files.Store, dirPath string, seq uint64, root bool) tea.Cmd {
	return func() tea.Msg {
		dir, err := readDir(ctx, store, dirPath)
		return dirLoadedMsg{Seq: seq, Root: root, Path: dirPath, Dir: dir, Err: err}
	}
}

// errNoStore is reported when a directory is read before a store is set.
var errNoStore = errors.New("store not set")

// readDir reads and sorts the entries of a directory: directories first, then
// by name.
func readDir(ctx context.Context, store files.Store, dirPath string) (*files.DirContext, error) {
	if store == nil {
		return nil, errNoStore
	}
	children, err := store.ReadDir(ctx, dirPath)
	if err != nil {
		return nil, err
	}
	dir := files.NewDirContext(store, dirPath, sortDirChildren(children))
	return dir, nil
}

// sortDirChildren sorts entries in place: directories first, then by name.
func sortDirChildren(children []os.DirEntry) []os.DirEntry {
	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir() != children[j].IsDir() {
			return children[i].IsDir()
		}
		return children[i].Name() < children[j].Name()
	})
	return children
}

// State persistence is done through these variables so tests can replace it.
var (
	saveCurrentDir      = ftstate.SaveCurrentDir
	saveSelectedTreeDir = ftstate.SaveSelectedTreeDir
	saveCurrentFileName = ftstate.SaveCurrentFileName
)
