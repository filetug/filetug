package tui

import (
	"maps"
	"os"
	"path"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/theme"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

const (
	dirEmoji = "📁"
	// rootID identifies the root node of the tree, whose paths are all
	// absolute and so can never equal it.
	rootID = "\x00root"
	// loadingID identifies the placeholder shown while a directory is read.
	loadingID = "\x00loading"
	// errorID identifies the row that shows why a directory could not be read.
	errorID = "\x00error"
)

// dirEmojis are the icons of well-known directories.
var dirEmojis = map[string]string{
	"library": "📚", "users": "👥", "applications": "🈸", "music": "🎹", "movies": "📺",
	"pictures": "🖼️", "desktop": "🖥️", "datatug": "🛥️", "documents": "🗃", "public": "📢",
	"temp": "⏳", "system": "🧠", "bin": "🚀", "sbin": "🚀", "private": "🔒",
}

func emojiForDir(name string) string {
	if emoji, ok := dirEmojis[strings.ToLower(name)]; ok {
		return emoji
	}
	return dirEmoji
}

func dirNodeText(name string) string { return emojiForDir(name) + " " + name }

// treeKeys are the tree's own bindings; ShortHelp lists them in the actions bar.
type treeKeys struct {
	Open, Parent, Find key.Binding
}

func defaultTreeKeys() treeKeys {
	return treeKeys{
		Open:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Parent: key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "parent")),
		Find:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("type", "find")),
	}
}

// treeScreen lists the sub-directories of one directory, the tree root. Moving
// the cursor previews a sub-directory in the file list, Enter makes it the new
// root, Left goes to the parent, and typing searches by name.
type treeScreen struct {
	sess     *session
	tree     widgets.Tree
	keys     treeKeys
	rootPath string
	children []os.DirEntry
	loading  bool
	err      error
	search   string
	// git is the styled git status of the root and of its sub-directories, by
	// path.
	git  map[string]string
	w, h int
}

func newTreeScreen(sess *session) treeScreen {
	return treeScreen{sess: sess, tree: widgets.NewTree("tree"), keys: defaultTreeKeys(), git: map[string]string{}}
}

var (
	_ nav.Screen       = treeScreen{}
	_ nav.Titled       = treeScreen{}
	_ nav.ShortHelper  = treeScreen{}
	_ widgets.Boundary = treeScreen{}
)

func (t treeScreen) Init() tea.Cmd { return nil }

// Title is the name of the root directory, or the search being typed.
func (t treeScreen) Title() string {
	if t.search != "" {
		return "Find: " + t.search
	}
	return t.panelTitle()
}

func (t treeScreen) panelTitle() string {
	root := t.sess.treeRoot
	if root == nil {
		return ""
	}
	storeRoot := t.sess.store.RootURL()
	if root.Path() == storeRoot.Path {
		return ""
	}
	trimmed := strings.TrimSuffix(root.Path(), "/")
	if storeRoot.Scheme == "file" && trimmed == userHomeDir {
		return "~"
	}
	_, title := path.Split(trimmed)
	return title
}

// userHomeDir is the title of the home directory in the tree.
var userHomeDir, _ = os.UserHomeDir()

func (t treeScreen) ShortHelp() []key.Binding {
	return []key.Binding{t.keys.Open, t.keys.Parent, t.keys.Find}
}

// AtEdge lets the shell move focus: Up leaves at the root row, Down at the last
// row, Right always goes on to the file list and Left never leaves.
func (t treeScreen) AtEdge(dir widgets.Direction) bool {
	switch dir {
	case widgets.Left:
		return false
	case widgets.Right:
		return true
	default:
		return t.tree.AtEdge(dir)
	}
}

func (t treeScreen) Update(msg tea.Msg) (nav.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.w, t.h = msg.Width, msg.Height
		t.tree.SetSize(t.w, t.h)
	case nav.ScreenFocusMsg:
		if msg.Focused {
			t.tree.Focus()
		} else {
			t.tree.Blur()
		}
	case goDirMsg:
		t = t.reset(msg)
	case dirLoadedMsg:
		t = t.loaded(msg)
	case gitStatusMsg:
		t = t.gitStatus(msg)
	case favoritesMsg:
		return t, nav.SetPanels(newFavoritesScreen(t.sess, t), nil, nav.FocusToMenu)
	case deleteMsg:
		return t, t.deleteCurrent()
	case widgets.NodeHighlightedMsg:
		return t, t.highlighted(msg.Node)
	case tea.KeyPressMsg:
		return t.key(msg)
	default:
		var cmd tea.Cmd
		t.tree, cmd = t.tree.Update(msg)
		return t, cmd
	}
	return t, nil
}

// reset shows the new root while its sub-directories are being read.
func (t treeScreen) reset(msg goDirMsg) treeScreen {
	t.rootPath = fsutils.ExpandHome(msg.Path)
	t.children, t.err, t.loading, t.search = nil, nil, true, ""
	t.git = map[string]string{}
	return t.rebuild(rootID)
}

// gitStatus shows the git status of a directory of the tree. Statuses of
// other lists, and stale ones, are not the tree's.
func (t treeScreen) gitStatus(msg gitStatusMsg) treeScreen {
	if msg.Scope != gitTree || msg.Seq != t.sess.rootSeq {
		return t
	}
	t.git = maps.Clone(t.git)
	t.git[msg.Path] = msg.Text
	return t.rebuild(t.current())
}

// loaded shows the sub-directories of the new root. Stale results and results
// for a directory shown without becoming the root are ignored.
func (t treeScreen) loaded(msg dirLoadedMsg) treeScreen {
	if msg.Seq != t.sess.seq || !msg.Root {
		return t
	}
	t.loading, t.err = false, msg.Err
	if msg.Err == nil {
		t.children = msg.Dir.Children()
	}
	return t.rebuild(rootID)
}

// rootText is the caption of the root row: the store root itself, or ".." to
// say that Enter goes up.
func (t treeScreen) rootText() string {
	root := t.sess.store.RootURL()
	if root.Path == "" {
		root.Path = "/"
	}
	if t.rootPath != root.Path {
		return ".."
	}
	if t.rootPath == "/" {
		return "/"
	}
	return strings.TrimSuffix(root.Path, "/")
}

// rebuild recreates the nodes from the state and puts the cursor on id.
func (t treeScreen) rebuild(id string) treeScreen {
	root := widgets.TreeNode{ID: rootID, Text: t.withGit(t.rootText(), t.rootPath), Ref: t.rootPath}
	switch {
	case t.loading:
		root.Children = []widgets.TreeNode{{ID: loadingID, Text: "Loading…", Color: theme.MutedColor(), Unselectable: true}}
	case t.err != nil:
		root.Children = []widgets.TreeNode{{ID: errorID, Text: dirEmoji + " " + t.err.Error(), Color: theme.ErrorColor(), Unselectable: true}}
	default:
		root.Children = t.nodes()
	}
	t.tree.SetRoots(root)
	t.tree.SetExpanded(rootID, true)
	t.tree.Select(id)
	return t
}

// nodes are the visible sub-directories, with the search match highlighted.
func (t treeScreen) nodes() []widgets.TreeNode {
	var nodes []widgets.TreeNode
	for _, child := range t.children {
		name := child.Name()
		if !child.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		nodes = append(nodes, widgets.TreeNode{
			ID:   path.Join(t.rootPath, name),
			Text: t.withGit(t.nodeText(name), path.Join(t.rootPath, name)),
			Ref:  path.Join(t.rootPath, name),
		})
	}
	return nodes
}

var matchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#90EE90"))

// nodeText is the caption of a sub-directory; the part matching the search is
// highlighted.
func (t treeScreen) nodeText(name string) string {
	text := dirNodeText(name)
	if t.search == "" {
		return text
	}
	i := strings.Index(strings.ToLower(name), t.search)
	if i < 0 {
		return text
	}
	match := name[i : i+len(t.search)]
	return dirEmoji + " " + name[:i] + matchStyle.Render(match) + name[i+len(t.search):]
}

// withGit appends the git status of a path to a caption.
func (t treeScreen) withGit(text, fullPath string) string {
	if status := t.git[fullPath]; status != "" {
		return text + " " + status
	}
	return text
}

// nodePath is the directory a node stands for.
func (t treeScreen) nodePath(id string) string {
	if id == rootID {
		return t.rootPath
	}
	return id
}

// highlighted previews the directory under the cursor in the file list.
func (t treeScreen) highlighted(node widgets.TreeNode) tea.Cmd {
	dir := t.nodePath(node.ID)
	saveSelectedTreeDir(dir)
	return widgets.Emit(showDirMsg{Path: dir})
}

// current is the id of the node under the cursor.
func (t treeScreen) current() string {
	node, ok := t.tree.Current()
	if !ok {
		return rootID
	}
	return node.ID
}

// deleteCurrent asks to delete the sub-directory under the cursor when the tree
// has the focus. The root row stands for the directory being listed and is not
// deleted from here.
func (t treeScreen) deleteCurrent() tea.Cmd {
	id := t.current()
	if !t.tree.Focused() || id == rootID || id == loadingID || id == errorID {
		return nil
	}
	return widgets.Emit(confirmDeleteMsg{Name: path.Base(id), Path: id})
}

// parentOf is the directory that contains dir.
func parentOf(dir string) string {
	parent, _ := path.Split(dir)
	return parent
}

func (t treeScreen) key(msg tea.KeyPressMsg) (nav.Screen, tea.Cmd) {
	switch msg.String() {
	case "left":
		return t, goDir(parentOf(t.nodePath(t.current())))
	case "right":
		return t, nav.SetFocus(nav.FocusToContent)
	case "enter":
		return t, t.open()
	case "backspace":
		return t.find(t.search[:max(len(t.search)-1, 0)])
	case "esc":
		return t.find("")
	}
	if text := msg.Key().Text; text != "" {
		return t.typed(text)
	}
	var cmd tea.Cmd
	t.tree, cmd = t.tree.Update(msg)
	return t, cmd
}

// typed handles a printable key: the global shortcuts, or the search.
func (t treeScreen) typed(text string) (nav.Screen, tea.Cmd) {
	switch text {
	case "/":
		return t, goDir("/")
	case "`":
		return t, goDir("~")
	case " ":
		if t.search == "" {
			return t, nil
		}
	}
	return t.find(t.search + strings.ToLower(text))
}

// open makes the directory under the cursor the root; on the root row it goes
// to the parent directory.
func (t treeScreen) open() tea.Cmd {
	id := t.current()
	dir := t.nodePath(id)
	if id == rootID {
		return goDir(parentOf(fsutils.ExpandHome(dir)))
	}
	return goDir(dir)
}

// find applies a search pattern and moves the cursor to the best match: a name
// that starts with the pattern, else one that contains it. A pattern that
// matches nothing loses its last character.
func (t treeScreen) find(pattern string) (nav.Screen, tea.Cmd) {
	for {
		t.search = pattern
		if pattern == "" {
			t = t.rebuild(rootID)
			return t, t.highlighted(widgets.TreeNode{ID: rootID})
		}
		t = t.rebuild(rootID)
		if id := t.bestMatch(); id != "" {
			t.tree.Select(id)
			return t, t.highlighted(widgets.TreeNode{ID: id})
		}
		pattern = pattern[:len(pattern)-1]
	}
}

// bestMatch is the id of the first node whose name starts with the search, or
// else contains it.
func (t treeScreen) bestMatch() string {
	contains := ""
	for _, child := range t.children {
		name := strings.ToLower(child.Name())
		if !child.IsDir() || strings.HasPrefix(name, ".") || !strings.Contains(name, t.search) {
			continue
		}
		id := path.Join(t.rootPath, child.Name())
		if strings.HasPrefix(name, t.search) {
			return id
		}
		if contains == "" {
			contains = id
		}
	}
	return contains
}

func (t treeScreen) View() string { return t.tree.View() }

// goDir returns the command that makes dir the root of the tree.
func goDir(dir string) tea.Cmd { return widgets.Emit(goDirMsg{Path: dir}) }
