package viewers

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/filetug/filetug/pkg/chroma2tcell"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var _ Previewer = (*TextPreviewer)(nil)

type TextPreviewer struct {
	*tview.TextView
	previewID uint64
}

func NewTextPreviewer() *TextPreviewer {
	return &TextPreviewer{
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetWrap(true).
			SetRegions(true).
			SetScrollable(true),
	}
}

func (p *TextPreviewer) Preview(entry files.EntryWithDirPath, data []byte, queueUpdateDraw func(func())) {
	if queueUpdateDraw == nil {
		queueUpdateDraw = func(f func()) { f() }
	}
	previewID := atomic.AddUint64(&p.previewID, 1)
	go func(previewID uint64) {
		if data == nil {
			var err error
			data, err = p.readFile(entry, 10*1024) // First 10KB
			if err != nil {
				errText := fmt.Sprintf("Failed to read file %s: %s", entry.FullName(), err.Error())
				queueUpdateDraw(func() {
					if !p.isCurrentPreview(previewID) {
						return
					}
					p.showError(errText)
				})
				return
			}
		}
		name := entry.Name()
		if lexer := lexers.Match(name); lexer == nil {
			queueUpdateDraw(func() {
				if !p.isCurrentPreview(previewID) {
					return
				}
				p.SetDynamicColors(false)
				p.SetText(string(data))
			})
		} else {
			colorized, err := chroma2tcell.Colorize(string(data), "dracula", lexer)
			queueUpdateDraw(func() {
				if !p.isCurrentPreview(previewID) {
					return
				}
				if err != nil {
					errText := err.Error()
					p.showError("Failed to format file: " + errText)
					return
				}
				p.Clear()
				p.SetDynamicColors(true)
				p.SetText(colorized)
				p.SetWrap(true)
			})
		}
	}(previewID)
}

func (p *TextPreviewer) Meta() tview.Primitive {
	return nil
}

func (p *TextPreviewer) Main() tview.Primitive {
	return p.TextView
}

func (p *TextPreviewer) readFile(entry files.EntryWithDirPath, max int) (data []byte, err error) {
	fullName := entry.FullName()
	data, err = fsutils.ReadFileData(fullName, max)
	if err != nil && !errors.Is(err, io.EOF) {
		return
	}
	return
}

func (p *TextPreviewer) isCurrentPreview(previewID uint64) bool {
	return atomic.LoadUint64(&p.previewID) == previewID
}

func (p *TextPreviewer) showError(text string) {
	p.SetDynamicColors(false)
	p.SetText(text)
	p.SetTextColor(tcell.ColorRed)
}
