package viewers

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/rivo/tview"
)

func TestNewMarkdownPreviewer(t *testing.T) {
	t.Parallel()
	p := NewMarkdownPreviewer(nil)
	assert.NotZero(t, p)
	var _ Previewer = p
}

// bumpingQueue returns a queueUpdateDraw that increments the previewer's
// previewID before invoking the callback, so the in-flight preview is stale by
// the time its isCurrentPreview check runs — exercising the stale `return`
// branches deterministically in synchronous mode.
func bumpingQueue(p **MarkdownPreviewer) func(func()) {
	return func(fn func()) {
		atomic.AddUint64(&(*p).previewID, 1)
		fn()
	}
}

func TestMarkdownPreviewer_PreviewSingle(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	SetTextPreviewerSyncForTest(true)
	t.Cleanup(func() { SetTextPreviewerSyncForTest(false) })

	immediate := func(fn func()) {
		if fn != nil {
			fn()
		}
	}

	// data provided → glamour renders → SetText carries the heading text.
	t.Run("renderProvidedData", func(t *testing.T) {
		p := NewMarkdownPreviewer(immediate)
		entry := files.NewEntryWithDirPath(testDirEntry{name: "readme.md"}, "/tmp")
		p.PreviewSingle(entry, []byte("# HeadingText\n\nbody"), nil)
		assert.Contains(t, p.GetText(false), "HeadingText")
	})

	// data nil + file on disk → readFile succeeds → renders.
	t.Run("readFromDisk", func(t *testing.T) {
		p := NewMarkdownPreviewer(immediate)
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "doc.md")
		assert.NoError(t, os.WriteFile(path, []byte("# DiskHeading"), 0644))
		entry := files.NewEntryWithDirPath(testDirEntry{name: "doc.md"}, tmpDir)
		p.PreviewSingle(entry, nil, nil)
		assert.Contains(t, p.GetText(false), "DiskHeading")
	})

	// data nil + missing file → readFile errors → showError with the error text.
	t.Run("readError", func(t *testing.T) {
		p := NewMarkdownPreviewer(immediate)
		tmpDir := t.TempDir()
		entry := files.NewEntryWithDirPath(testDirEntry{name: "missing.md"}, tmpDir)
		p.PreviewSingle(entry, nil, nil)
		assert.Contains(t, p.GetText(false), "missing.md")
	})

	// glamour render error → showError("Failed to render markdown: ...").
	t.Run("renderError", func(t *testing.T) {
		orig := glamourRender
		glamourRender = func(string, string) (string, error) { return "", errors.New("boom") }
		t.Cleanup(func() { glamourRender = orig })
		p := NewMarkdownPreviewer(immediate)
		entry := files.NewEntryWithDirPath(testDirEntry{name: "x.md"}, "/tmp")
		p.PreviewSingle(entry, []byte("# x"), nil)
		assert.Contains(t, p.GetText(false), "Failed to render markdown")
	})

	// nil queueUpdateDraw (direct struct literal) → run() returns early.
	t.Run("nilQueueUpdateDraw", func(t *testing.T) {
		p := &MarkdownPreviewer{TextPreviewer: TextPreviewer{TextView: tview.NewTextView()}}
		entry := files.NewEntryWithDirPath(testDirEntry{name: "x.md"}, "/tmp")
		p.PreviewSingle(entry, []byte("# data"), nil)
		assert.Equal(t, "", p.GetText(false))
	})

	// Stale variants: the callback runs after previewID advanced → each `return`.
	t.Run("staleReadError", func(t *testing.T) {
		var p *MarkdownPreviewer
		p = NewMarkdownPreviewer(bumpingQueue(&p))
		tmpDir := t.TempDir()
		entry := files.NewEntryWithDirPath(testDirEntry{name: "missing.md"}, tmpDir)
		p.PreviewSingle(entry, nil, nil)
		assert.Equal(t, "", p.GetText(false)) // showError skipped (stale)
	})

	t.Run("staleRenderError", func(t *testing.T) {
		orig := glamourRender
		glamourRender = func(string, string) (string, error) { return "", errors.New("boom") }
		t.Cleanup(func() { glamourRender = orig })
		var p *MarkdownPreviewer
		p = NewMarkdownPreviewer(bumpingQueue(&p))
		entry := files.NewEntryWithDirPath(testDirEntry{name: "x.md"}, "/tmp")
		p.PreviewSingle(entry, []byte("# x"), nil)
		assert.Equal(t, "", p.GetText(false))
	})

	t.Run("staleSuccess", func(t *testing.T) {
		var p *MarkdownPreviewer
		p = NewMarkdownPreviewer(bumpingQueue(&p))
		entry := files.NewEntryWithDirPath(testDirEntry{name: "x.md"}, "/tmp")
		p.PreviewSingle(entry, []byte("# HeadingText"), nil)
		assert.Equal(t, "", p.GetText(false)) // SetText skipped (stale)
	})
}

// TestMarkdownPreviewer_Async exercises the asynchronous `go run` path (no sync
// seam).
func TestMarkdownPreviewer_Async(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	p := NewMarkdownPreviewer(func(fn func()) {
		fn()
		close(done)
	})
	entry := files.NewEntryWithDirPath(testDirEntry{name: "async.md"}, "/tmp")
	p.PreviewSingle(entry, []byte("# AsyncHeading"), nil)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async preview")
	}
	assert.Contains(t, p.GetText(false), "AsyncHeading")
}
