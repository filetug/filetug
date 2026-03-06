package viewers

import (
	"sync/atomic"

	"github.com/charmbracelet/glamour"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/rivo/tview"
)

var _ Previewer = (*MarkdownPreviewer)(nil)

type MarkdownPreviewer struct {
	TextPreviewer
}

func NewMarkdownPreviewer(queueUpdateDraw navigator.UpdateDrawQueuer) *MarkdownPreviewer {
	return &MarkdownPreviewer{
		TextPreviewer: *NewTextPreviewer(queueUpdateDraw),
	}
}

func (p *MarkdownPreviewer) PreviewSingle(entry files.EntryWithDirPath, data []byte, dataErr error) {
	previewID := atomic.AddUint64(&p.previewID, 1)
	run := func(previewID uint64) {
		if p.queueUpdateDraw == nil {
			return
		}
		if data == nil {
			var err error
			data, err = p.readFile(entry, 0)
			if err != nil {
				errText := err.Error()
				p.queueUpdateDraw(func() {
					if !p.isCurrentPreview(previewID) {
						return
					}
					p.showError(errText)
				})
				return
			}
		}
		rendered, err := glamour.Render(string(data), "dark")
		if err != nil {
			errText := err.Error()
			p.queueUpdateDraw(func() {
				if !p.isCurrentPreview(previewID) {
					return
				}
				p.showError("Failed to render markdown: " + errText)
			})
			return
		}
		translated := tview.TranslateANSI(rendered)
		p.queueUpdateDraw(func() {
			if !p.isCurrentPreview(previewID) {
				return
			}
			p.SetDynamicColors(true)
			p.SetText(translated)
		})
	}
	if textPreviewerSyncForTest {
		run(previewID)
		return
	}
	go run(previewID)
}
