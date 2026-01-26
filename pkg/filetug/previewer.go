package filetug

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/filetug/filetug/pkg/chroma2tcell"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/filetug/filetug/pkg/sneatv"
	"github.com/filetug/filetug/pkg/viewers"
	"github.com/filetug/filetug/pkg/viewers/imageviewer"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/strongo/dsstore"
)

type previewer struct {
	*sneatv.Boxed
	flex       *tview.Flex
	nav        *Navigator
	attributes *tview.Table
	separator  *tview.TextView
	textView   *tview.TextView
}

func newPreviewer(nav *Navigator) *previewer {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	p := previewer{
		Boxed: sneatv.NewBoxed(
			flex,
			sneatv.WithLeftBorder(0, -1),
		),
		flex:       flex,
		attributes: tview.NewTable(),
		separator:  tview.NewTextView().SetText(strings.Repeat("─", 20)).SetTextColor(tcell.ColorGray),
		textView:   tview.NewTextView(),
		nav:        nav,
	}

	p.textView.SetWrap(false)
	p.textView.SetDynamicColors(true)
	p.textView.SetText("To be implemented.")
	p.textView.SetFocusFunc(func() {
		nav.activeCol = 2
	})

	p.flex.AddItem(p.attributes, 2, 0, false)
	p.flex.AddItem(p.separator, 1, 0, false)
	p.flex.AddItem(p.textView, 0, 1, false)

	p.flex.SetFocusFunc(func() {
		nav.activeCol = 2
		p.flex.SetBorderColor(sneatv.CurrentTheme.FocusedBorderColor)
	})
	nav.previewerFocusFunc = func() {
		nav.activeCol = 2
		p.flex.SetBorderColor(sneatv.CurrentTheme.FocusedBorderColor)
	}
	p.flex.SetBlurFunc(func() {
		p.flex.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	})
	nav.previewerBlurFunc = func() {
		p.flex.SetBorderColor(sneatv.CurrentTheme.BlurredBorderColor)
	}

	p.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			nav.setAppFocus(nav.files)
			return nil
		case tcell.KeyUp:
			nav.o.moveFocusUp(p.textView)
			return nil
		default:
			return event
		}
	})

	return &p
}

func (p *previewer) SetErr(err error) {
	p.textView.Clear()
	p.textView.SetDynamicColors(true)
	p.textView.SetText(err.Error())
	p.textView.SetTextColor(tcell.ColorRed)
}

func (p *previewer) SetText(text string) {
	p.textView.Clear()
	p.textView.SetDynamicColors(true)
	p.textView.SetText(text)
	p.textView.SetTextColor(tcell.ColorWhiteSmoke)
}

func (p *previewer) readFile(fullName string, max int) (data []byte, err error) {
	data, err = fsutils.ReadFileData(fullName, max)
	if err != nil && !errors.Is(err, io.EOF) {
		p.textView.SetText(fmt.Sprintf("Failed to read file %s: %s", fullName, err.Error()))
		p.textView.SetTextColor(tcell.ColorRed)
		return
	}
	return
}

func (p *previewer) PreviewFile(entry files.EntryWithDirPath) {
	name := entry.Name()
	fullName := entry.Path()
	if name == "" {
		_, name = path.Split(fullName)
	}
	p.SetTitle(name)
	var data []byte
	var err error
	switch name {
	case ".DS_Store":
		data, err = p.readFile(fullName, 0)
		if err != nil {
			return
		}
		bufferRead := bytes.NewBuffer(data)
		var s dsstore.Store
		err = s.Read(bufferRead)
		if err != nil {
			p.SetErr(err)
			return
		}
		var sb strings.Builder
		for _, r := range s.Records {
			sb.WriteString(fmt.Sprintf("%s: %s\n", r.FileName, r.Type))
		}
		data = []byte(sb.String())
	default:
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".json":
			data, err = p.readFile(fullName, 0)
			if err != nil {
				return
			}
			str, err := prettyJSON(string(data))
			if err == nil {
				data = []byte(str)
			}
		case ".log":
			data, err = p.readFile(fullName, -1024)
		case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".riff", ".tiff", ".vp8", ".webp":
			meta := imageviewer.ImagePreviewer{}.GetMeta(fullName)
			if meta != nil {
				metaTable := viewers.NewMetaTable()
				metaTable.SetMeta(meta)
				p.nav.right.SetContent(metaTable)
			}
			return
		}
	}
	lexer := lexers.Match(name)
	if data == nil && err == nil {
		data, err = p.readFile(fullName, 1024*1024)
		if err != nil && !errors.Is(err, io.EOF) {
			return
		}
	}
	if lexer == nil {
		p.textView.Clear()
		p.textView.SetDynamicColors(true)
		p.textView.SetWrap(false)
		p.textView.SetText(string(data))
		p.nav.previewer.textView.SetTextColor(tcell.ColorWhiteSmoke)
		return
	}
	colorized, err := chroma2tcell.Colorize(string(data), "dracula", lexer)
	if err != nil {
		p.textView.Clear()
		p.textView.SetDynamicColors(true)
		p.textView.SetText("Failed to format file: " + err.Error())
		p.textView.SetTextColor(tcell.ColorRed)
		return
	}
	p.textView.Clear()
	p.textView.SetDynamicColors(true)
	p.textView.SetText(colorized)
	p.textView.SetWrap(true)
	//p.textView.SetTextColor(tcell.ColorWhiteSmoke)
}

func prettyJSON(input string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(input), "", "  ") // 2-space indent
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
