package filetug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/blacktop/go-termimg"
	"github.com/datatug/filetug/pkg/chroma2tcell"
	"github.com/datatug/filetug/pkg/fileviewers/dsstore"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type previewer struct {
	*tview.Flex
	nav      *Navigator
	textView *tview.TextView
}

func newPreviewer(nav *Navigator) *previewer {
	p := previewer{
		Flex: tview.NewFlex(),
		nav:  nav,
	}
	p.SetTitle("Preview")
	p.SetBorder(true)
	p.SetBorderColor(Style.BlurBorderColor)

	p.textView = tview.NewTextView()
	p.textView.SetWrap(false)
	p.textView.SetDynamicColors(true)
	p.textView.SetText("To be implemented.")
	p.textView.SetFocusFunc(func() {
		nav.activeCol = 2
	})

	p.AddItem(p.textView, 0, 1, false)

	p.SetFocusFunc(func() {
		nav.activeCol = 2
		p.SetBorderColor(Style.FocusedBorderColor)
		//nav.app.SetFocus(tv)
	})
	nav.previewerFocusFunc = func() {
		nav.activeCol = 2
		p.SetBorderColor(Style.FocusedBorderColor)
	}
	p.SetBlurFunc(func() {
		p.SetBorderColor(Style.BlurBorderColor)
	})
	nav.previewerBlurFunc = func() {
		p.SetBorderColor(Style.BlurBorderColor)
	}

	p.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			nav.app.SetFocus(nav.files)
			return nil
		case tcell.KeyUp:
			nav.o.moveFocusUp(p)
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

func (p *previewer) PreviewFile(name, fullName string) {
	data, err := os.ReadFile(fullName)
	if err != nil {
		p.textView.SetText(fmt.Sprintf("Error reading file %s: %s", fullName, err.Error()))
		p.textView.SetTextColor(tcell.ColorRed)
		return
	}
	if name == "" {
		_, name = path.Split(fullName)
	}
	switch name {
	case ".DS_Store":
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
			str, err := prettyJSON(string(data))
			if err == nil {
				data = []byte(str)
			}
		case ".png", ".jpg", ".jpeg", ".gif":
			p.textView.Clear()
			p.textView.SetDynamicColors(true)
			_, _, w, h := p.textView.GetInnerRect()
			if w == 0 || h == 0 {
				w, h = 80, 40 // Fallback
			}
			img, err := termimg.Open(fullName)
			if err != nil {
				p.SetErr(err)
				return
			}
			rendered, err := img.Width(w).Height(h).Render()
			if err != nil {
				p.SetErr(err)
				return
			}
			p.textView.SetWrap(false)
			writer := tview.ANSIWriter(p.textView)
			_, _ = writer.Write([]byte(rendered))
			return
		}
	}
	lexer := lexers.Match(name)
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
