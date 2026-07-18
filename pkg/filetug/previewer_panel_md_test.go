package filetug

import (
	"testing"

	"github.com/filetug/filetug/pkg/viewers"
)

// TestPreviewerPanel_GetFilePreviewer_Markdown covers the ".md" dispatch branch
// of getFilePreviewer: both the "no cached markdown previewer → create new" path
// and the "cached markdown previewer → reuse" path.
func TestPreviewerPanel_GetFilePreviewer_Markdown(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	panel := newPreviewerPanel(nav)

	// No cached markdown previewer yet → a fresh one is created.
	pv := panel.getFilePreviewer("readme.md")
	if _, ok := pv.(*viewers.MarkdownPreviewer); !ok {
		t.Fatalf("getFilePreviewer(.md): expected *MarkdownPreviewer, got %T", pv)
	}

	// With a cached markdown previewer → the same instance is returned.
	cached := viewers.NewMarkdownPreviewer(nav.app.QueueUpdateDraw)
	panel.previewer = cached
	if got := panel.getFilePreviewer("doc.md"); got != cached {
		t.Error("getFilePreviewer(.md): expected the cached markdown previewer to be reused")
	}
}
