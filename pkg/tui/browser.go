package tui

import (
	"path"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/nav"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

// pane is one of the two panes of the browser.
type pane int

const (
	paneFiles pane = iota
	panePreview
)

// crumbsID is the id the shell gives its breadcrumbs.
const crumbsID = "nav.crumbs"

// defaultProportions are the widths of the file list and the preview.
var defaultProportions = [2]int{12, 7}

// browser is the content of the shell: the file list next to the preview of the
// entry under its cursor. It owns navigation: it starts every read and drops
// the results that are no longer wanted.
type browser struct {
	sess    *session
	files   filesPane
	preview previewPane
	// focus is the pane that has the focus while the browser is focused.
	focus   pane
	focused bool
	weights [2]int
	w, h    int
	// startMsg is the navigation that opens the application.
	startMsg tea.Msg
	// crumbPaths are the directories the breadcrumbs after the first stand for.
	crumbPaths []string
}

func newBrowser(sess *session) browser {
	return browser{sess: sess, files: newFilesPane(), preview: newPreviewPane(), weights: defaultProportions}
}

var (
	_ nav.Screen       = browser{}
	_ nav.Borderless   = browser{}
	_ nav.ShortHelper  = browser{}
	_ widgets.Boundary = browser{}
)

// Init opens the starting directory.
func (b browser) Init() tea.Cmd { return widgets.Emit(b.startMsg) }

// Borderless implements nav.Borderless: the browser draws the frames of its
// panes itself.
func (b browser) Borderless() bool { return true }

// ShortHelp lists the keys of the file list in the actions bar.
func (b browser) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	}
}

// AtEdge lets the shell move focus at the edges of the focused pane.
func (b browser) AtEdge(dir widgets.Direction) bool {
	switch {
	case dir == widgets.Right:
		return false
	case dir == widgets.Left:
		return b.focus == paneFiles
	case b.focus == paneFiles:
		return b.files.AtEdge(dir)
	default:
		return b.preview.AtEdge(dir)
	}
}

// geometry is the outer width of the file list and the preview, with the one
// cell gap between them left out.
func (b browser) geometry() (filesW, previewW int) {
	parts := widgets.Split(max(b.w-1, 0), widgets.Fill(b.weights[0]), widgets.Fill(b.weights[1]))
	return parts[0], parts[1]
}

// layout tells the panes their inner size.
func (b *browser) layout() {
	fw, pw := b.geometry()
	frame := widgets.NewFrame()
	w, h := frame.Inner(fw, b.h)
	b.files.SetSize(w, h)
	w, h = frame.Inner(pw, b.h)
	b.preview.SetSize(w, h)
}

// applyFocus tells the panes whether they have the focus.
func (b *browser) applyFocus() {
	if b.focused && b.focus == panePreview {
		b.preview.Focus()
	} else {
		b.preview.Blur()
	}
}

func (b browser) Update(msg tea.Msg) (nav.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.w, b.h = msg.Width, msg.Height
		b.layout()
	case nav.ScreenFocusMsg:
		b.focused = msg.Focused
		b.applyFocus()
	case goDirMsg:
		return b.goDir(msg)
	case showDirMsg:
		return b.showDir(msg)
	case dirLoadedMsg:
		return b.loaded(msg)
	case previewMsg:
		if msg.Seq == b.sess.previewSeq {
			b.preview.Show(msg)
		}
	case grid.SelectionChangedMsg:
		if msg.ID == filesID {
			cmd := b.previewCurrent()
			return b, cmd
		}
	case grid.RowActivatedMsg:
		return b, b.activated(msg)
	case widgets.CrumbSelectedMsg:
		return b, b.crumbSelected(msg)
	case resizeMsg:
		b = b.resize(msg)
	case tea.KeyPressMsg:
		return b.key(msg)
	case tea.MouseMsg:
		return b.mouse(msg)
	}
	return b, nil
}

// goDir makes a directory the root of the tree and starts reading it.
func (b browser) goDir(msg goDirMsg) (nav.Screen, tea.Cmd) {
	s := b.sess
	if msg.Store != nil {
		s.store = msg.Store
	}
	dirPath := fsutils.ExpandHome(msg.Path)
	s.treeRoot = files.NewDirContext(s.store, dirPath, nil)
	s.current = s.treeRoot
	ctx, seq := s.begin()
	root := s.store.RootURL()
	saveCurrentDir(root.String(), msg.Path)
	b.preview.Loading("")
	crumbs := b.crumbsCmd()
	return b, tea.Batch(loadDirCmd(ctx, s.store, dirPath, seq, true), crumbs)
}

// showDir shows a directory in the file list while the tree keeps its root.
func (b browser) showDir(msg showDirMsg) (nav.Screen, tea.Cmd) {
	s := b.sess
	dirPath := fsutils.ExpandHome(msg.Path)
	if s.currentPath() == dirPath {
		return b, nil
	}
	s.current = files.NewDirContext(s.store, dirPath, nil)
	ctx, seq := s.begin()
	crumbs := b.crumbsCmd()
	return b, tea.Batch(loadDirCmd(ctx, s.store, dirPath, seq, false), crumbs)
}

// loaded shows a directory that has been read.
func (b browser) loaded(msg dirLoadedMsg) (nav.Screen, tea.Cmd) {
	if msg.Seq != b.sess.seq {
		return b, nil
	}
	if msg.Err != nil {
		b.files.SetError(msg.Err)
		b.preview.Show(previewMsg{Title: path.Base(msg.Path), Err: msg.Err})
		return b, nil
	}
	showDirs := msg.Path != b.sess.treeRoot.Path()
	b.files.SetDir(msg.Dir, showDirs, b.sess.entryName)
	cmd := b.previewCurrent()
	return b, cmd
}

// previewCurrent starts the preview of the entry under the cursor.
func (b *browser) previewCurrent() tea.Cmd {
	ref, ok := b.files.CurrentRef()
	if !ok {
		b.preview.Show(previewMsg{})
		return nil
	}
	s := b.sess
	if !ref.IsDir {
		s.entryName = ref.Entry.Name()
		saveCurrentFileName(s.entryName)
	}
	s.previewSeq++
	b.preview.Loading(ref.Entry.Name())
	return previewCmd(s.previewSeq, s.store, ref.Entry, ref.IsDir, b.preview.w)
}

// activated opens the directory of an activated row.
func (b browser) activated(msg grid.RowActivatedMsg) tea.Cmd {
	ref, ok := msg.Row.Ref.(rowRef)
	if msg.ID != filesID || !ok || !ref.IsDir {
		return nil
	}
	return goDir(ref.Entry.FullName())
}

// crumbSelected opens the directory of an activated breadcrumb.
func (b browser) crumbSelected(msg widgets.CrumbSelectedMsg) tea.Cmd {
	i := msg.Index - 1
	if msg.ID != crumbsID || i < 0 || i >= len(b.crumbPaths) {
		return nil
	}
	return goDir(b.crumbPaths[i])
}

// crumbsCmd sets the breadcrumbs to the path of the current directory.
func (b *browser) crumbsCmd() tea.Cmd {
	crumbs, paths := crumbsFor(b.sess.store, b.sess.currentPath())
	b.crumbPaths = paths
	return nav.SetBreadcrumbs(crumbs...)
}

// crumbsFor builds the breadcrumbs of a directory: the application, the root
// of the store, and one crumb per directory below it, with the path each opens.
func crumbsFor(store files.Store, current string) (crumbs []widgets.Crumb, paths []string) {
	crumbs = []widgets.Crumb{{Title: "FileTug"}}
	if store == nil {
		return crumbs, nil
	}
	rootPath := store.RootURL().Path
	if rootPath == "" {
		rootPath = "/"
	}
	crumbs = append(crumbs, widgets.Crumb{Title: strings.TrimSuffix(store.RootTitle(), "/")})
	paths = []string{rootPath}
	rel := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(current, rootPath), "/"), "/")
	if current == "" || rel == "" {
		return crumbs, paths
	}
	elems := []string{rootPath}
	for _, name := range strings.Split(rel, "/") {
		elems = append(elems, name)
		if name == "" {
			name = "{EMPTY PATH ITEM}"
		}
		crumbs = append(crumbs, widgets.Crumb{Title: name})
		paths = append(paths, path.Join(elems...))
	}
	return crumbs, paths
}

// key routes a key to the focused pane. Right moves from the file list to the
// preview and Left back; Space and the shortcuts are the grid's.
func (b browser) key(msg tea.KeyPressMsg) (nav.Screen, tea.Cmd) {
	switch {
	case b.focus == paneFiles && msg.String() == "right":
		return b.focusPane(panePreview), nil
	case b.focus == panePreview && msg.String() == "left":
		return b.focusPane(paneFiles), nil
	case b.focus == paneFiles:
		var cmd tea.Cmd
		b.files, cmd = b.files.Update(msg)
		return b, cmd
	}
	var cmd tea.Cmd
	b.preview, cmd = b.preview.Update(msg)
	return b, cmd
}

// focusPane moves the focus to a pane.
func (b browser) focusPane(p pane) browser {
	b.focus = p
	b.applyFocus()
	return b
}

// mouse focuses the pane that is clicked and scrolls the one under the wheel.
func (b browser) mouse(msg tea.MouseMsg) (nav.Screen, tea.Cmd) {
	fw, _ := b.geometry()
	target := panePreview
	if msg.Mouse().X < fw {
		target = paneFiles
	}
	if _, ok := msg.(tea.MouseClickMsg); ok {
		b = b.focusPane(target)
		return b, nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		return b.wheel(target, wheel)
	}
	return b, nil
}

// wheel scrolls a pane: the preview text, or the file list by cursor.
func (b browser) wheel(target pane, msg tea.MouseWheelMsg) (nav.Screen, tea.Cmd) {
	if target == panePreview {
		var cmd tea.Cmd
		b.preview, cmd = b.preview.Update(msg)
		return b, cmd
	}
	name := "down"
	if msg.Button == tea.MouseWheelUp {
		name = "up"
	}
	var cmd tea.Cmd
	b.files, cmd = b.files.Update(keyPress(name))
	return b, cmd
}

// keyPress builds the key press of a named key.
func keyPress(name string) tea.KeyPressMsg {
	switch name {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	}
	return tea.KeyPressMsg{Code: tea.KeyDown}
}

// resizeMsg changes the share of the file list and the preview, as the Alt+plus,
// Alt+minus and Alt+0 keys do. A zero Delta restores the defaults.
type resizeMsg struct{ Delta int }

func (b browser) resize(msg resizeMsg) browser {
	if msg.Delta == 0 {
		b.weights = defaultProportions
	} else {
		which := int(b.focus)
		other := 1 - which
		if b.weights[which]+msg.Delta >= 1 && b.weights[other]-msg.Delta >= 1 {
			b.weights[which] += msg.Delta
			b.weights[other] -= msg.Delta
		}
	}
	b.layout()
	return b
}

// filesTitle is the name of the directory the file list shows.
func (b browser) filesTitle() string {
	name := b.sess.currentPath()
	if name == "" {
		return ""
	}
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "" {
		return "/"
	}
	return path.Base(trimmed)
}

func (b browser) View() string {
	if b.w <= 0 || b.h <= 0 {
		return ""
	}
	fw, pw := b.geometry()
	left := widgets.NewFrame().WithTitle(b.filesTitle()).WithFocus(b.focused && b.focus == paneFiles).
		Render(b.files.View(b.focused && b.focus == paneFiles), fw, b.h)
	right := widgets.NewFrame().WithTitle(b.preview.Title()).WithFocus(b.focused && b.focus == panePreview).
		Render(b.preview.View(), pw, b.h)
	gap := strings.TrimSuffix(strings.Repeat(" \n", b.h), "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
}
