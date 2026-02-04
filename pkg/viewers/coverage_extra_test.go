package viewers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type testDirEntry struct {
	name  string
	isDir bool
}

func (e testDirEntry) Name() string               { return e.name }
func (e testDirEntry) IsDir() bool                { return e.isDir }
func (e testDirEntry) Type() os.FileMode          { return 0 }
func (e testDirEntry) Info() (os.FileInfo, error) { return nil, nil }

type testTabsApp struct {
	queueUpdateDraw func(func())
	setFocus        func(tview.Primitive)
}

func (a *testTabsApp) QueueUpdateDraw(f func()) {
	if a.queueUpdateDraw != nil {
		a.queueUpdateDraw(f)
		return
	}
	if f != nil {
		f()
	}
}

func (a *testTabsApp) SetFocus(p tview.Primitive) {
	if a.setFocus != nil {
		a.setFocus(p)
	}
}

func TestMockPreviewer_Coverage(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mock := NewMockPreviewer(ctrl)
	tv := tview.NewTextView()

	mock.EXPECT().Main().Return(tv)
	mock.EXPECT().Meta().Return(tv)
	mock.EXPECT().PreviewSingle(gomock.Any(), gomock.Any(), gomock.Nil())

	assert.Equal(t, tv, mock.Main())
	assert.Equal(t, tv, mock.Meta())
	entry := files.NewEntryWithDirPath(testDirEntry{name: "file.txt"}, "/tmp")
	mock.PreviewSingle(entry, []byte("data"), nil)
}

func TestTextPreviewer_PreviewSingleBranches(t *testing.T) {
	t.Parallel()
	SetTextPreviewerSyncForTest(true)
	t.Cleanup(func() { SetTextPreviewerSyncForTest(false) })

	t.Run("readError", func(t *testing.T) {
		p := NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		tmpDir := t.TempDir()
		entry := files.NewEntryWithDirPath(testDirEntry{name: "missing.txt"}, tmpDir)
		p.PreviewSingle(entry, nil, nil)
		assert.Contains(t, p.GetText(false), "Failed to read file")
	})

	t.Run("dataErrNoLexer", func(t *testing.T) {
		p := NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		entry := files.NewEntryWithDirPath(testDirEntry{name: "file.unknown"}, "/tmp")
		dataErr := errors.New("data err")
		data := []byte("data err\nhello world")
		p.PreviewSingle(entry, data, dataErr)
		text := p.GetText(true)
		assert.Contains(t, text, "data err")
		assert.Contains(t, text, "hello world")
	})

	t.Run("dataErrWithLexer", func(t *testing.T) {
		p := NewTextPreviewer(func(f func()) {
			if f != nil {
				f()
			}
		})
		entry := files.NewEntryWithDirPath(testDirEntry{name: "file.go"}, "/tmp")
		dataErr := errors.New("partial")
		data := []byte("partial\npackage main\n")
		p.PreviewSingle(entry, data, dataErr)
		text := p.GetText(true)
		assert.Contains(t, text, "partial")
	})

	t.Run("nilQueueUpdateDraw", func(t *testing.T) {
		p := &TextPreviewer{TextView: tview.NewTextView()}
		entry := files.NewEntryWithDirPath(testDirEntry{name: "file.txt"}, "/tmp")
		p.PreviewSingle(entry, []byte("data"), nil)
	})
}

func TestGitDirStatusPreviewer_RenderEntriesAndSetMessage(t *testing.T) {
	t.Parallel()
	p := NewGitDirStatusPreviewer()
	p.entries = []gitDirStatusEntry{
		{displayName: "added.txt", staged: true, badge: gitBadge{text: "A", color: tcell.ColorGreen}},
		{displayName: "modified.txt", staged: false, badge: gitBadge{text: "M", color: tcell.ColorYellow}},
	}
	p.renderEntries()
	assert.Equal(t, "✓ added.txt", p.table.GetCell(0, 0).Text)
	assert.Equal(t, "modified.txt", p.table.GetCell(1, 0).Text[2:])

	p.setMessage("hello", tcell.ColorRed)
	assert.Equal(t, "hello", p.table.GetCell(0, 0).Text)

	pNil := &GitDirStatusPreviewer{}
	pNil.setMessage("ignored", tcell.ColorRed)
	pNil.renderEntries()
}

func TestDirPreviewer_setTabsCoverage(t *testing.T) {
	t.Parallel()
	app := &testTabsApp{}
	p := NewDirPreviewer(app)
	p.setTabs(true)
	assert.NotNil(t, p.tabs)
	p.setTabs(false)
	assert.NotNil(t, p.tabs)
}
