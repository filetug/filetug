package previewers

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/riff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

var _ Previewer = (*ImagePreviewer)(nil)

type ImagePreviewer struct {
}

func (i ImagePreviewer) GetMeta(path string) (meta *Meta) {
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
	main := MetaGroup{
		ID:    "main",
		Title: "Format: " + strings.ToUpper(format),
	}
	main.Records = append(main.Records,
		&MetaRecord{
			ID:         "width",
			Title:      "Width",
			Value:      strconv.Itoa(cfg.Width),
			ValueAlign: AlignRight,
		},
		&MetaRecord{
			ID:         "height",
			Title:      "Height",
			Value:      strconv.Itoa(cfg.Height),
			ValueAlign: AlignRight,
		},
	)
	return &Meta{
		Groups: []*MetaGroup{
			&main,
		},
	}
}
