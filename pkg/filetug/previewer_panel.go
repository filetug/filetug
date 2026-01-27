package filetug

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/filetug/filetug/pkg/viewers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type previewerPanel struct {
	*sneatv.Boxed
	rows      *tview.Flex
	nav       *Navigator
	attrsRow  *tview.Flex
	fsAttrs   *tview.Table
	sizeCell  *tview.TableCell
	modCell   *tview.TableCell
	separator *tview.TextView
	previewer viewers.Previewer
	textView  *tview.TextView
}

func newPreviewerPanel(nav *Navigator) *previewerPanel {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	separator := tview.NewTextView()
	separator.SetText(strings.Repeat("─", 20))
	separator.SetTextColor(tcell.ColorGray)
	p := previewerPanel{
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(0, -1),
		),
		rows:      flex,
		attrsRow:  tview.NewFlex(),
		separator: separator,
		textView:  tview.NewTextView(),
		nav:       nav,
	}
	p.attrsRow.SetDirection(tview.FlexRow)
	p.fsAttrs = p.createAttrsTable()
	p.fsAttrs.SetSelectable(true, true)

	p.textView.SetWrap(false)
	p.textView.SetDynamicColors(true)
	p.textView.SetText("To be implemented.")
	p.textView.SetFocusFunc(func() {
		nav.activeCol = 2
	})

	p.attrsRow.AddItem(p.fsAttrs, 0, 1, false)

	p.rows.AddItem(p.attrsRow, 2, 0, false)
	p.rows.AddItem(p.separator, 1, 0, false)
	//p.rows.AddItem(p.textView, 0, 1, false)

	p.rows.SetFocusFunc(func() {
		nav.activeCol = 2
		p.rows.SetBorderColor(sneatv.CurrentTheme.FocusedBorderColor)
	})
	nav.previewerFocusFunc = func() {
		nav.activeCol = 2
		p.rows.SetBorderColor(sneatv.CurrentTheme.FocusedBorderColor)
	}
	p.rows.SetBlurFunc(func() {
		p.rows.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	})
	nav.previewerBlurFunc = func() {
		p.rows.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	}

	p.rows.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			nav.setAppFocus(nav.files)
			return nil
		case tcell.KeyUp:
			//nav.o.moveFocusUp(p.fsAttrs)
			//return nil
			return event
		default:
			return event
		}
	})

	return &p
}

func (p *previewerPanel) createAttrsTable() *tview.Table {
	t := tview.NewTable()
	sizeLabelCell := tview.NewTableCell("Size")
	sizeLabelCell.SetAlign(tview.AlignRight)
	sizeLabelCell.SetTextColor(sneatv.CurrentTheme.LabelColor)
	sizeLabelCell.SetSelectable(false)
	t.SetCell(0, 0, sizeLabelCell)
	p.sizeCell = tview.NewTableCell("")
	t.SetCell(0, 1, p.sizeCell)
	modLabelCell := tview.NewTableCell("Modified")
	modLabelCell.SetAlign(tview.AlignRight)
	modLabelCell.SetTextColor(sneatv.CurrentTheme.LabelColor)
	modLabelCell.SetSelectable(false)
	t.SetCell(1, 0, modLabelCell)
	p.modCell = tview.NewTableCell("")
	p.modCell.SetAlign(tview.AlignRight)
	t.SetCell(1, 1, p.modCell)
	return t
}

func (p *previewerPanel) setPreviewer(previewer viewers.Previewer) {
	if p.previewer != nil {
		if meta := p.previewer.Meta(); meta != nil {
			p.attrsRow.RemoveItem(meta)
		}
		if main := p.previewer.Main(); main != nil {
			p.rows.RemoveItem(main)
		}
	}
	p.previewer = previewer
	if previewer != nil {
		//if meta := previewer.Meta(); meta != nil {
		//	p.attrsRow.AddItem(meta, 0, 1, false)
		//}
		if main := previewer.Main(); main != nil {
			p.rows.AddItem(main, 0, 1, false)
		}
	}
}

func (p *previewerPanel) SetErr(err error) {
	p.textView.Clear()
	p.textView.SetDynamicColors(true)
	errText := err.Error()
	p.textView.SetText(errText)
	p.textView.SetTextColor(tcell.ColorRed)
}

func (p *previewerPanel) SetText(text string) {
	p.textView.Clear()
	p.textView.SetDynamicColors(true)
	p.textView.SetText(text)
	p.textView.SetTextColor(tcell.ColorWhiteSmoke)
}

func (p *previewerPanel) PreviewEntry(entry files.EntryWithDirPath) {
	if info, err := entry.Info(); err == nil {
		size := info.Size()
		sizeText := fsutils.GetSizeShortText(size)
		p.sizeCell.SetText(sizeText)
		modTime := info.ModTime()
		p.modCell.SetText(modTime.Format(time.RFC3339))
	}

	name := entry.Name()
	fullName := entry.FullName()
	if name == "" {
		_, name = path.Split(fullName)
	}
	p.SetTitle(name)
	var previewer viewers.Previewer
	switch name {
	case ".DS_Store":
		if _, ok := p.previewer.(*viewers.DsstorePreviewer); !ok {
			previewer = viewers.NewDsstorePreviewer()
		}
	default:
		nameExt := filepath.Ext(name)
		ext := strings.ToLower(nameExt)
		switch ext {
		case ".json":
			if _, ok := p.previewer.(*viewers.JsonPreviewer); !ok {
				previewer = viewers.NewJsonPreviewer()
				p.setPreviewer(previewer)
			}
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".riff", ".tiff", ".vp8", ".webp":
			if _, ok := p.previewer.(*viewers.ImagePreviewer); !ok {
				previewer = viewers.NewImagePreviewer()
			}
		default:
			if _, ok := p.previewer.(*viewers.TextPreviewer); !ok {
				previewer = viewers.NewTextPreviewer()
			}
		}
	}
	if previewer != nil {
		p.setPreviewer(previewer)
	}
	if p.previewer != nil {
		p.previewer.Preview(entry, nil, p.nav.queueUpdateDraw)
	}
}
