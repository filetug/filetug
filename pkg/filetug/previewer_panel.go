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
	"github.com/strongo/strongo-tui/pkg/colors"
	"github.com/strongo/strongo-tui/pkg/themes"
)

type previewerPanel struct {
	*sneatv.Boxed
	app          PreviewerApp
	rows         *tview.Flex
	nav          *Navigator
	attrsRow     *tview.Flex
	fsAttrs      *tview.Table
	sizeCell     *tview.TableCell
	modCell      *tview.TableCell
	separator    *tview.TextView
	previewer    viewers.Previewer
	textView     *tview.TextView
	dirPreviewer *viewers.DirPreviewer
}

func newPreviewerPanel(nav *Navigator) *previewerPanel {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	separator := tview.NewTextView()
	separator.SetText(strings.Repeat("─", 20))
	separator.SetTextColor(tcell.ColorGray)

	filterSetter := viewers.WithDirSummaryFilterSetter(nav.files.SetFilter)
	focusLeft := viewers.WithDirSummaryFocusLeft(func() {
		nav.app.SetFocus(nav.files)
	})
	queueUpdateDraw := viewers.WithDirSummaryQueueUpdateDraw(nav.app.QueueUpdateDraw)
	p := previewerPanel{
		app: nav.app,
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(0, -1),
		),
		dirPreviewer: viewers.NewDirPreviewer(nav.app, filterSetter, focusLeft, queueUpdateDraw),
		rows:         flex,
		attrsRow:     tview.NewFlex(),
		separator:    separator,
		textView:     tview.NewTextView(),
		nav:          nav,
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
		p.rows.SetBorderColor(themes.CurrentTheme.FocusedBorderColor())
	})
	nav.previewerFocusFunc = func() {
		nav.activeCol = 2
		p.rows.SetBorderColor(themes.CurrentTheme.FocusedBorderColor())
	}
	p.rows.SetBlurFunc(func() {
		p.rows.SetBorderColor(themes.CurrentTheme.BlurredBorderColor())
	})
	nav.previewerBlurFunc = func() {
		p.rows.SetBorderColor(themes.CurrentTheme.BlurredBorderColor())
	}

	p.rows.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			nav.app.SetFocus(nav.files)
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
	sizeLabelCell.SetTextColor(themes.CurrentTheme.LabelColor())
	sizeLabelCell.SetSelectable(false)
	t.SetCell(0, 0, sizeLabelCell)
	p.sizeCell = tview.NewTableCell("")
	t.SetCell(0, 1, p.sizeCell)
	modLabelCell := tview.NewTableCell("Modified")
	modLabelCell.SetAlign(tview.AlignRight)
	modLabelCell.SetTextColor(themes.CurrentTheme.LabelColor())
	modLabelCell.SetSelectable(false)
	t.SetCell(1, 0, modLabelCell)
	p.modCell = tview.NewTableCell("")
	p.modCell.SetAlign(tview.AlignRight)
	t.SetCell(1, 1, p.modCell)
	return t
}

func (p *previewerPanel) setPreviewer(previewer viewers.Previewer) {
	if p.previewer == previewer {
		return
	}
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
	p.textView.SetTextColor(colors.TableHeaderColor)
}

func (p *previewerPanel) PreviewEntry(entry files.EntryWithDirPath) {
	info, err := entry.Info()
	if err == nil && info != nil {
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

	if entry.IsDir() {
		previewer = p.dirPreviewer
	} else {
		previewer = p.getFilePreviewer(name)
	}

	p.setPreviewer(previewer)
	p.previewer.PreviewSingle(entry, nil, nil)
}

func (p *previewerPanel) getFilePreviewer(name string) viewers.Previewer {
	switch name {
	case ".DS_Store":
		if dsStorePreviewer, ok := p.previewer.(*viewers.DsstorePreviewer); ok {
			return dsStorePreviewer
		}
		return viewers.NewDsstorePreviewer(p.app.QueueUpdateDraw)
	default:
		nameExt := filepath.Ext(name)
		ext := strings.ToLower(nameExt)
		switch ext {
		case ".json", ".jsonb":
			if jsonPreviewer, ok := p.previewer.(*viewers.JsonPreviewer); ok {
				return jsonPreviewer
			}
			return viewers.NewJsonPreviewer(p.app.QueueUpdateDraw)
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".riff", ".tiff", ".vp8", ".webp":
			if imagePreviewer, ok := p.previewer.(*viewers.ImagePreviewer); ok {
				return imagePreviewer
			}
			return viewers.NewImagePreviewer(p.app.QueueUpdateDraw)
		default:
			if textPreviewer, ok := p.previewer.(*viewers.TextPreviewer); ok {
				return textPreviewer
			}
			return viewers.NewTextPreviewer(p.app.QueueUpdateDraw)
		}
	}
}
