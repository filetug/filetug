package tui

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/files/ftpfile"
	"github.com/filetug/filetug/pkg/files/httpfile"
	"github.com/filetug/filetug/pkg/files/osfile"
	"github.com/filetug/filetug/pkg/filetug/ftstate"
	"github.com/tuigoff/tuigoff/pkg/nav"
)

// Options configure the application.
type Options struct {
	// Path is the directory, or the file, to open; empty restores the location
	// of the previous run.
	Path string
}

// Seams over the environment, replaced by tests.
var (
	getState  = ftstate.GetState
	absPath   = filepath.Abs
	statPath  = os.Stat
	newOSFile = func() files.Store { return osfile.NewStore("/") }
	newHTTP   = func(u url.URL) files.Store { return httpfile.NewStore(u) }
	newFTP    = func(u url.URL) files.Store {
		if s := ftpfile.NewStore(u); s != nil {
			return s
		}
		return nil
	}
)

// start is where the application opens.
type start struct {
	store files.Store
	dir   string
	// entry is the file to select in the directory.
	entry string
}

// startFor decides where to open: the given path, or the persisted location.
func startFor(opts Options) start {
	if opts.Path == "" {
		return persistedStart()
	}
	st := start{store: newOSFile(), dir: opts.Path}
	abs, err := absPath(opts.Path)
	if err != nil {
		return st
	}
	st.dir = abs
	if info, err := statPath(abs); err == nil && !info.IsDir() {
		st.dir, st.entry = filepath.Dir(abs), filepath.Base(abs)
	}
	return st
}

// persistedStart restores the store and directory saved by the previous run.
func persistedStart() start {
	st := start{store: newOSFile(), dir: "~"}
	state, err := getState()
	if state == nil {
		return st
	}
	storeURL := state.Store
	if storeURL == "" {
		storeURL = "file:"
	}
	scheme, _, _ := strings.Cut(storeURL, ":")
	if root, perr := url.Parse(storeURL); perr == nil {
		switch scheme {
		case "http", "https":
			st.store = newHTTP(*root)
		case "ftp":
			if s := newFTP(*root); s != nil {
				st.store = s
			}
		}
	}
	if state.CurrentDir != "" {
		st.dir = state.CurrentDir
	}
	if strings.HasPrefix(st.dir, "https://") {
		current, perr := url.Parse(st.dir)
		if perr != nil {
			return st
		}
		st.dir = current.Path
		current.Path = "/"
		st.store = newHTTP(*current)
	}
	if err == nil {
		st.entry = state.CurrentDirEntry
	}
	return st
}

// rootPage is the page the application opens: the directory tree in the menu
// and the file list with the preview as content.
func rootPage(opts Options) nav.Page {
	st := startFor(opts)
	sess := &session{store: st.store, entryName: st.entry}
	browse := newBrowser(sess)
	browse.startMsg = goDirMsg{Store: st.store, Path: st.dir}
	return nav.Page{Title: "FileTug", Menu: newTreeScreen(sess), Content: browse, Focus: nav.FocusToMenu}
}

// shellOptions configure the navigation shell.
func shellOptions() []nav.Option {
	return []nav.Option{nav.WithoutLogin(), nav.WithActions(actions()...)}
}

// New creates the application: the directory tree, the file list and the
// preview inside the navigation shell.
func New(opts Options) nav.Model { return nav.New(rootPage(opts), shellOptions()...) }

// Run starts the application and returns when it ends.
func Run(opts Options, options ...tea.ProgramOption) error {
	return runShell(New(opts), options...)
}

// runShell is the seam through which the shell is run.
var runShell = func(m nav.Model, options ...tea.ProgramOption) error { return nav.Run(m, options...) }

// macOptionRunes maps the letters produced by Option+key on a US Mac keyboard
// back to their key, for terminals that send the character instead of Alt+key.
var macOptionRunes = map[rune]rune{
	'å': 'a', '∫': 'b', 'ç': 'c', '∂': 'd', 'ƒ': 'f', '©': 'g', '˙': 'h', '∆': 'j', '˚': 'k',
	'¬': 'l', 'µ': 'm', 'ø': 'o', 'π': 'p', 'œ': 'q', '®': 'r', 'ß': 's', '†': 't', '√': 'v',
	'∑': 'w', '≈': 'x', '¥': 'y', 'Ω': 'z', 'º': '0', '–': '-', '≠': '=', '÷': '/',
}

// altKeys are the key names that trigger Alt+r: the Alt chord itself and the
// character a Mac sends for Option+r.
func altKeys(r rune) []string {
	keys := []string{"alt+" + string(r)}
	for option, base := range macOptionRunes {
		if base == r {
			keys = append(keys, string(option))
		}
	}
	return keys
}

func altBinding(r rune, help string, extra ...rune) key.Binding {
	keys := altKeys(r)
	for _, e := range extra {
		keys = append(keys, altKeys(e)...)
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp("alt+"+string(r), help))
}

// actions are the keys that work everywhere in the application.
func actions() []nav.Action {
	return []nav.Action{
		{ID: "root", Binding: altBinding('/', "root", 'r'), Msg: goDirMsg{Path: "/"}},
		{ID: "home", Binding: altBinding('~', "home", '`', 'h'), Msg: goDirMsg{Path: "~"}},
		{ID: "exit", Binding: altBinding('x', "exit"), Msg: tea.QuitMsg{}},
		{ID: "grow", Binding: altBinding('=', "wider", '+'), Msg: resizeMsg{Delta: 1}, Hidden: true},
		{ID: "shrink", Binding: altBinding('-', "narrower", '_'), Msg: resizeMsg{Delta: -1}, Hidden: true},
		{ID: "reset", Binding: altBinding('0', "reset sizes"), Msg: resizeMsg{}, Hidden: true},
	}
}
