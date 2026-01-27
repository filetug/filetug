package viewers

import (
	"bytes"
	"encoding/json"

	"github.com/filetug/filetug/pkg/files"
)

var _ Previewer = (*JsonPreviewer)(nil)

type JsonPreviewer struct {
	TextPreviewer
}

func NewJsonPreviewer() *JsonPreviewer {
	textPreviewer := NewTextPreviewer()
	return &JsonPreviewer{
		TextPreviewer: *textPreviewer,
	}
}

func (p JsonPreviewer) Preview(entry files.EntryWithDirPath, data []byte, queueUpdateDraw func(func())) {
	if data == nil {
		var err error
		data, err = p.readFile(entry, 0)
		if err != nil {
			return
		}
	}
	dataText := string(data)
	str, _ := prettyJSON(dataText)
	data = []byte(str)
	p.TextPreviewer.Preview(entry, data, queueUpdateDraw)
}

func prettyJSON(input string) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(input), "", "  ") // 2-space indent
	if err != nil {
		return input, err
	}
	return out.String(), nil
}
