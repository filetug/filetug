package tui

import (
	"path"
	"path/filepath"
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
	paneWorktrees
)

// crumbsID is the id the shell gives its breadcrumbs.
const crumbsID = "nav.crumbs"

// defaultProportions are the widths of the file list and the preview.
var defaultProportions = [2]int{12, 7}

// browser is the content of the shell: the file list next to the preview of the
// entry under its cursor. It owns navigation: it starts every read and drops
// the results that are no longer wanted.
type browser struct {
	sess      *session
	files     filesPane
	preview   previewPane
	worktrees worktreesPane
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
	return browser{sess: sess, files: newFilesPane(), preview: newPreviewPane(), worktrees: newWorktreesPane(), weights: defaultProportions}
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
	case b.focus == paneWorktrees:
		return dir == widgets.Down && b.worktrees.AtEdge(dir)
	case dir == widgets.Down && b.worktrees.visible:
		return false
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
	previewH, worktreesH := b.rightHeights()
	w, h = frame.Inner(pw, previewH)
	b.preview.SetSize(w, h)
	w, h = frame.Inner(pw, worktreesH)
	b.worktrees.SetSize(w, h)
}

// rightHeights are the outer heights of the preview and the worktrees pane,
// which shares the right column when it is open.
func (b browser) rightHeights() (previewH, worktreesH int) {
	if !b.worktrees.visible {
		return b.h, 0
	}
	parts := widgets.Split(b.h, widgets.Fill(1), widgets.Fill(1))
	return parts[0], parts[1]
}

// applyFocus tells the panes whether they have the focus.
func (b *browser) applyFocus() {
	if b.focused && b.focus == panePreview {
		b.preview.Focus()
	} else {
		b.preview.Blur()
	}
	if b.focused && b.focus == paneWorktrees {
		b.worktrees.Focus()
	} else {
		b.worktrees.Blur()
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
	case gitStatusMsg:
		return b.gitStatus(msg)
	case worktreesToggleMsg:
		return b.toggleWorktrees()
	case worktreeProbeMsg:
		return b.probed(msg)
	case worktreesListedMsg:
		cmd := b.worktrees.Listed(msg)
		return b, cmd
	case worktreesEnrichedMsg:
		b.worktrees.Enriched(msg)
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
	ctx, seq := s.begin(true)
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
	ctx, seq := s.begin(false)
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
	preview := b.previewCurrent()
	return b, tea.Batch(preview, b.gitCmds(msg), b.probeCmd(msg.Path))
}

// gitCmds start reading the git status of the entries of a directory that has
// been read and, when it became the tree root, of the directories in the tree.
// Only the local file system is searched for repositories.
func (b browser) gitCmds(msg dirLoadedMsg) tea.Cmd {
	s := b.sess
	if s.store.RootURL().Scheme != "file" {
		return nil
	}
	entries := msg.Dir.Entries()
	reqs := make([]gitReq, 0, len(entries))
	var dirs []gitReq
	for _, e := range entries {
		isDir := b.files.state.rows.entryIsDir(e)
		reqs = append(reqs, gitReq{Path: e.FullName(), IsDir: isDir})
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, gitReq{Path: e.FullName(), IsDir: true})
		}
	}
	cmd := gitStatusCmd(s.listCtx, s.seq, gitFiles, msg.Path, reqs)
	if !msg.Root {
		return cmd
	}
	dirs = append([]gitReq{{Path: msg.Path, IsDir: true}}, dirs...)
	return tea.Batch(cmd, gitStatusCmd(s.rootCtx, s.rootSeq, gitTree, msg.Path, dirs))
}

// gitStatus applies the status of a file list entry and asks for the next
// status of the stream. A status read for a directory that is no longer shown
// ends the stream.
func (b browser) gitStatus(msg gitStatusMsg) (nav.Screen, tea.Cmd) {
	if msg.Scope == gitFiles {
		if msg.Seq != b.sess.seq {
			return b, nil
		}
		b.files.SetGitText(msg.Path, msg.Text)
	} else if msg.Seq != b.sess.rootSeq {
		return b, nil
	}
	return b, waitGit(msg.stream)
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
	if item, ok := msg.Row.Ref.(worktreeInfo); ok && msg.ID == worktreesID {
		if item.Prunable {
			return nil
		}
		return goDir(item.Path)
	}
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
	case b.focus == paneWorktrees:
		return b.worktreesKey(msg)
	case b.focus == panePreview && msg.String() == "down" && b.worktrees.visible && b.preview.AtEdge(widgets.Down):
		return b.focusPane(paneWorktrees), nil
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

// worktreesKey handles a key while the worktrees pane has the focus: Esc closes
// it, Left and Up at the top return to the file list and the preview, r reads
// the worktrees again.
func (b browser) worktreesKey(msg tea.KeyPressMsg) (nav.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		b.worktrees.Hide()
		b.layout()
		return b.focusPane(paneFiles), nil
	case "left":
		return b.focusPane(paneFiles), nil
	case "up":
		if b.worktrees.AtEdge(widgets.Up) {
			return b.focusPane(panePreview), nil
		}
	case "r", "R":
		cmd := b.worktrees.Load(b.worktrees.repoRoot)
		return b, cmd
	}
	var cmd tea.Cmd
	b.worktrees, cmd = b.worktrees.Update(msg)
	return b, cmd
}

// probeCmd looks up the repository of a directory that has been shown, to open
// the worktrees pane at a repository root or to keep an open pane in step.
func (b browser) probeCmd(dirPath string) tea.Cmd {
	if b.sess.store.RootURL().Scheme != "file" {
		return nil
	}
	if b.worktrees.visible {
		return probeWorktreesCmd(dirPath, probeSync)
	}
	return probeWorktreesCmd(dirPath, probeAuto)
}

// toggleWorktrees is Alt+W: focus the pane if it is open, else open it for the
// repository of the current directory.
func (b browser) toggleWorktrees() (nav.Screen, tea.Cmd) {
	if b.sess.store == nil || b.sess.store.RootURL().Scheme != "file" {
		return b, nil
	}
	if b.worktrees.visible && b.worktrees.repoRoot != "" {
		return b.focusPane(paneWorktrees), nav.SetFocus(nav.FocusToContent)
	}
	path := b.sess.currentPath()
	if path == "" {
		return b, nil
	}
	return b, probeWorktreesCmd(path, probeShow)
}

// probed opens, updates or closes the worktrees pane for the repository found
// for a directory.
func (b browser) probed(msg worktreeProbeMsg) (nav.Screen, tea.Cmd) {
	if msg.Path != b.sess.currentPath() {
		return b, nil
	}
	var cmd tea.Cmd
	switch {
	case msg.Intent == probeShow && msg.RepoRoot == "":
		b.worktrees.ShowNoRepository()
	case msg.Intent == probeShow:
		b.worktrees.visible = true
		cmd = tea.Batch(b.worktrees.EnsureLoaded(msg.RepoRoot), nav.SetFocus(nav.FocusToContent))
		b = b.focusPane(paneWorktrees)
	case msg.Intent == probeSync && msg.RepoRoot == "":
		b.worktrees.Hide()
		if b.focus == paneWorktrees {
			b = b.focusPane(paneFiles)
		}
	case msg.Intent == probeSync:
		cmd = b.worktrees.EnsureLoaded(msg.RepoRoot)
	case msg.RepoRoot != "" && filepath.Clean(msg.RepoRoot) == filepath.Clean(msg.Path):
		b.worktrees.visible = true
		cmd = b.worktrees.EnsureLoaded(msg.RepoRoot)
	}
	b.layout()
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
	m := msg.Mouse()
	target := panePreview
	if m.X < fw {
		target = paneFiles
	} else if previewH, _ := b.rightHeights(); b.worktrees.visible && m.Y >= previewH {
		target = paneWorktrees
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
	if target == paneWorktrees {
		return b.worktreesKey(keyPress(wheelKey(msg)))
	}
	var cmd tea.Cmd
	b.files, cmd = b.files.Update(keyPress(wheelKey(msg)))
	return b, cmd
}

// wheelKey is the key a wheel turn stands for.
func wheelKey(msg tea.MouseWheelMsg) string {
	if msg.Button == tea.MouseWheelUp {
		return "up"
	}
	return "down"
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
		which := min(int(b.focus), 1)
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
	previewH, worktreesH := b.rightHeights()
	right := widgets.NewFrame().WithTitle(b.preview.Title()).WithFocus(b.focused && b.focus == panePreview).
		Render(b.preview.View(), pw, previewH)
	if b.worktrees.visible {
		below := widgets.NewFrame().WithTitle("Worktrees").WithFocus(b.focused && b.focus == paneWorktrees).
			Render(b.worktrees.View(), pw, worktreesH)
		right = right + "\n" + below
	}
	gap := strings.TrimSuffix(strings.Repeat(" \n", b.h), "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)
}
