package viewers

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/rivo/tview"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/riff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

var _ Previewer = (*ImagePreviewer)(nil)

type ImagePreviewer struct {
	metaTable       *MetaTable
	queueUpdateDraw func(func())
}

func NewImagePreviewer(queueUpdateDraw func(func())) *ImagePreviewer {
	previewer := &ImagePreviewer{
		metaTable:       NewMetaTable(),
		queueUpdateDraw: queueUpdateDraw,
	}
	previewer.metaTable.SetSelectable(true, true)
	return previewer
}

func (p ImagePreviewer) PreviewSingle(entry files.EntryWithDirPath, _ []byte, _ error) {
	go func() {
		fullName := entry.FullName()
		meta := p.GetMeta(fullName)
		if meta != nil {
			p.queueUpdateDraw(func() {
				p.metaTable.SetMeta(meta)
			})
		}
	}()
}

func (p ImagePreviewer) Meta() tview.Primitive {
	return nil
}

func (p ImagePreviewer) Main() tview.Primitive {
	return p.metaTable
}

func (p ImagePreviewer) GetMeta(path string) (meta *Meta) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return
	}
	upperFormat := strings.ToUpper(format)
	main := MetaGroup{
		ID:    "main",
		Title: "Format: " + upperFormat,
	}
	widthValue := strconv.Itoa(cfg.Width)
	heightValue := strconv.Itoa(cfg.Height)
	widthRecord := &MetaRecord{
		ID:         "width",
		Title:      "Width",
		Value:      widthValue,
		ValueAlign: AlignRight,
	}
	heightRecord := &MetaRecord{
		ID:         "height",
		Title:      "Height",
		Value:      heightValue,
		ValueAlign: AlignRight,
	}
	main.Records = append(main.Records, widthRecord, heightRecord)
	return &Meta{
		Groups: []*MetaGroup{
			&main,
		},
	}
}
