// Package charmui contains Filetug's Bubble Tea MVP.
package charmui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/tree"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"github.com/strongo/strongo-tui/charm"
)

const maxPreviewBytes = 128 * 1024

// DirectoryStore is the small, UI-independent I/O boundary used by the MVP.
// It deliberately exposes only the local-directory journey.
type DirectoryStore interface {
	ReadDir(context.Context, string) ([]os.DirEntry, error)
}

type fileReader func(context.Context, string, int64) (string, error)

type previewRenderer func(string, string) (string, error)

type focusPanel uint8

const (
	focusTree focusPanel = iota
	focusFiles
	focusPreview
)

type directoryLoadedMsg struct {
	path    string
	request uint64
	entries []os.DirEntry
	err     error
}

type previewLoadedMsg struct {
	path    string
	request uint64
	content string
	err     error
}

// Model owns every piece of Bubble Tea MVP state. Asynchronous commands only
// return identity-bearing messages; they never mutate this state directly.
type Model struct {
	store    DirectoryStore
	readFile fileReader
	render   previewRenderer

	rootPath    string
	currentPath string
	entries     []os.DirEntry
	selected    int
	listOffset  int
	focus       focusPanel

	width   int
	height  int
	tree    tree.Model
	preview viewport.Model
	styles  styleAdapter

	dirRequest     uint64
	previewRequest uint64
	dirCancel      context.CancelFunc
	previewCancel  context.CancelFunc
	directoryState string
	previewState   string
	err            string
}

// NewModel creates the MVP against path. It does not read the filesystem until
// Init, keeping construction deterministic for direct model tests.
func NewModel(path string, store DirectoryStore) (*Model, error) {
	if store == nil {
		return nil, errors.New("directory store is required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve initial path: %w", err)
	}
	rootNode := tree.Root(absolutePath)
	treeModel := tree.New(rootNode, 1, 1)
	treeModel.SetShowHelp(false)
	previewModel := viewport.New(viewport.WithWidth(1), viewport.WithHeight(1))
	previewModel.SoftWrap = true
	return &Model{
		store:          store,
		readFile:       readLocalFile,
		render:         renderPreview,
		rootPath:       absolutePath,
		currentPath:    absolutePath,
		tree:           treeModel,
		preview:        previewModel,
		styles:         newStyleAdapter(charm.DefaultTheme()),
		directoryState: "loading",
		previewState:   "Select a file to preview",
	}, nil
}

// Init starts the first asynchronous directory read.
func (m *Model) Init() tea.Cmd {
	return m.loadDirectory(m.currentPath)
}

// Update owns window sizing, focus routing, and stale-result rejection.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(typed.Width, typed.Height)
	case tea.KeyPressMsg:
		return m.updateKey(typed)
	case directoryLoadedMsg:
		if typed.request != m.dirRequest || typed.path != m.currentPath {
			return m, nil
		}
		m.acceptDirectory(typed)
		if typed.err != nil || len(m.entries) == 0 {
			return m, nil
		}
		return m, m.loadPreviewForSelection()
	case previewLoadedMsg:
		if typed.request != m.previewRequest || typed.path != m.selectedPath() {
			return m, nil
		}
		m.acceptPreview(typed)
	}
	return m, nil
}

func (m *Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "q", "esc", "ctrl+c":
		m.cancelRequests()
		return m, tea.Quit
	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, nil
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
		return m, nil
	}

	switch m.focus {
	case focusTree:
		return m.updateTree(msg)
	case focusFiles:
		return m.updateFiles(key)
	case focusPreview:
		updatedPreview, cmd := m.preview.Update(msg)
		m.preview = updatedPreview
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateTree(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" || key == "right" || key == "l" {
		node := m.tree.NodeAtCurrentOffset()
		if node != nil {
			if path, ok := node.GivenValue().(string); ok {
				return m, m.loadDirectory(path)
			}
		}
	}
	updatedTree, cmd := m.tree.Update(msg)
	m.tree = updatedTree
	return m, cmd
}

func (m *Model) updateFiles(key string) (tea.Model, tea.Cmd) {
	if key == "left" || key == "h" || key == "backspace" {
		if m.currentPath != m.rootPath {
			parent := filepath.Dir(m.currentPath)
			if isPathWithinRoot(parent, m.rootPath) {
				return m, m.loadDirectory(parent)
			}
		}
	}
	if len(m.entries) == 0 {
		return m, nil
	}
	previous := m.selected
	switch key {
	case "down", "j":
		if m.selected < len(m.entries)-1 {
			m.selected++
		}
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "pgdown":
		m.selected = minInt(len(m.entries)-1, m.selected+m.listHeight())
	case "pgup":
		m.selected = maxInt(0, m.selected-m.listHeight())
	case "enter", "right", "l":
		entry := m.entries[m.selected]
		if entry.IsDir() {
			path := filepath.Join(m.currentPath, entry.Name())
			return m, m.loadDirectory(path)
		}
	}
	m.ensureSelectedVisible()
	if previous != m.selected {
		return m, m.loadPreviewForSelection()
	}
	return m, nil
}

func (m *Model) loadDirectory(path string) tea.Cmd {
	m.cancelDirectory()
	m.cancelPreview()
	m.dirRequest++
	m.previewRequest++
	request := m.dirRequest
	m.currentPath = path
	m.entries = nil
	m.selected = 0
	m.listOffset = 0
	m.directoryState = "loading"
	m.previewState = "Loading directory…"
	m.preview.SetContent(m.previewState)
	m.err = ""
	m.rebuildTree()
	ctx, cancel := context.WithCancel(context.Background())
	m.dirCancel = cancel
	return func() tea.Msg {
		entries, err := m.store.ReadDir(ctx, path)
		return directoryLoadedMsg{path: path, request: request, entries: entries, err: err}
	}
}

func (m *Model) loadPreviewForSelection() tea.Cmd {
	m.cancelPreview()
	m.previewRequest++
	request := m.previewRequest
	path := m.selectedPath()
	if path == "" {
		m.previewState = "No file selected"
		return nil
	}
	entry := m.entries[m.selected]
	if entry.IsDir() {
		m.previewState = "Directory selected — press enter to open"
		m.preview.SetContent(m.previewState)
		return nil
	}
	m.previewState = "Loading preview…"
	m.preview.SetContent(m.previewState)
	ctx, cancel := context.WithCancel(context.Background())
	m.previewCancel = cancel
	return func() tea.Msg {
		content, err := m.readFile(ctx, path, maxPreviewBytes)
		if err != nil {
			return previewLoadedMsg{path: path, request: request, err: err}
		}
		content, err = m.render(path, content)
		return previewLoadedMsg{path: path, request: request, content: content, err: err}
	}
}

func (m *Model) acceptDirectory(msg directoryLoadedMsg) {
	if msg.err != nil {
		m.entries = nil
		m.selected = 0
		m.listOffset = 0
		m.directoryState = "error"
		m.err = msg.err.Error()
		m.previewState = "Directory error: " + m.err
		m.preview.SetContent(m.previewState)
		m.rebuildTree()
		return
	}
	sort.SliceStable(msg.entries, func(i, j int) bool {
		left := msg.entries[i]
		right := msg.entries[j]
		if left.IsDir() != right.IsDir() {
			return left.IsDir()
		}
		return strings.ToLower(left.Name()) < strings.ToLower(right.Name())
	})
	m.entries = msg.entries
	m.selected = 0
	m.listOffset = 0
	if len(m.entries) == 0 {
		m.directoryState = "empty"
		m.previewState = "Directory is empty"
		m.preview.SetContent(m.previewState)
	} else {
		m.directoryState = "ready"
	}
	m.rebuildTree()
}

func (m *Model) acceptPreview(msg previewLoadedMsg) {
	if msg.err != nil {
		m.previewState = "Preview error: " + msg.err.Error()
		m.preview.SetContent(m.previewState)
		return
	}
	m.previewState = "ready"
	m.preview.SetContent(msg.content)
}

func (m *Model) rebuildTree() {
	root := tree.Root(m.rootPath)
	currentNode := root
	if m.currentPath != m.rootPath {
		relativePath, err := filepath.Rel(m.rootPath, m.currentPath)
		if err == nil && relativePath != "." && !strings.HasPrefix(relativePath, "..") {
			parts := strings.Split(relativePath, string(filepath.Separator))
			path := m.rootPath
			for _, part := range parts {
				path = filepath.Join(path, part)
				child := tree.Root(part)
				child.SetValue(path)
				child.Open()
				currentNode.Child(child)
				currentNode = child
			}
		}
	}
	for _, entry := range m.entries {
		if entry.IsDir() {
			path := filepath.Join(m.currentPath, entry.Name())
			child := tree.Root(entry.Name())
			child.SetValue(path)
			currentNode.Child(child)
		}
	}
	m.tree.SetNodes(root)
	m.tree.SetSize(m.treeContentWidth(), m.panelBodyHeight())
	m.selectCurrentTreeNode()
}

func (m *Model) selectCurrentTreeNode() {
	for _, node := range m.tree.AllNodes() {
		path, ok := node.GivenValue().(string)
		if ok && path == m.currentPath {
			m.tree.SetYOffset(node.YOffset())
			return
		}
	}
}

func (m *Model) setSize(width, height int) {
	m.width = maxInt(1, width)
	m.height = maxInt(1, height)
	m.tree.SetSize(m.treeContentWidth(), m.panelBodyHeight())
	m.preview.SetWidth(m.previewContentWidth())
	m.preview.SetHeight(m.panelBodyHeight())
	m.ensureSelectedVisible()
}

func (m *Model) cancelDirectory() {
	if m.dirCancel != nil {
		m.dirCancel()
		m.dirCancel = nil
	}
}

func (m *Model) cancelPreview() {
	if m.previewCancel != nil {
		m.previewCancel()
		m.previewCancel = nil
	}
}

func (m *Model) cancelRequests() {
	m.cancelDirectory()
	m.cancelPreview()
}

func (m *Model) selectedPath() string {
	if m.selected < 0 || m.selected >= len(m.entries) {
		return ""
	}
	return filepath.Join(m.currentPath, m.entries[m.selected].Name())
}

func (m *Model) listHeight() int {
	return maxInt(1, m.panelBodyHeight())
}

func (m *Model) panelContentHeight() int {
	return maxInt(1, m.height-4)
}

func (m *Model) panelBodyHeight() int {
	return maxInt(1, m.panelContentHeight()-1)
}

func (m *Model) treeContentWidth() int {
	return maxInt(1, m.panelLayout().treeOuter-m.styles.panelFrameWidth())
}

func (m *Model) previewContentWidth() int {
	return maxInt(1, m.panelLayout().previewOuter-m.styles.panelFrameWidth())
}

func (m *Model) ensureSelectedVisible() {
	if m.selected < m.listOffset {
		m.listOffset = m.selected
	}
	last := m.listOffset + m.listHeight() - 1
	if m.selected > last {
		m.listOffset = m.selected - m.listHeight() + 1
	}
}

func readLocalFile(ctx context.Context, path string, limit int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()
	reader := io.LimitReader(file, limit+1)
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	if int64(len(content)) > limit {
		content = append(content[:limit], []byte("\n\n[preview truncated]")...)
	}
	return string(content), nil
}

func renderPreview(path, content string) (string, error) {
	if strings.EqualFold(filepath.Ext(path), ".md") || strings.EqualFold(filepath.Ext(path), ".markdown") {
		rendered, err := glamour.Render(content, "dark")
		if err != nil {
			return "", err
		}
		return rendered, nil
	}
	return content, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func isPathWithinRoot(path, root string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relativePath == "." || !strings.HasPrefix(relativePath, "..")
}

var _ tea.Model = (*Model)(nil)
