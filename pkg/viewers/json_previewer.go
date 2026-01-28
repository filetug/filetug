package viewers

import (
	"bytes"
	"encoding/json"
	"fmt"

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

func (p JsonPreviewer) Preview(entry files.EntryWithDirPath, data []byte, dataErr error, queueUpdateDraw func(func())) {
	if data == nil {
		var err error
		data, err = p.readFile(entry, 0)
		if err != nil {
			return
		}
	}
	formatted, err := prettyJSON(data)
	if err != nil {
		errText := err.Error()
		prefix := "Invalid JSON: " + errText + "\n"
		data = append([]byte(prefix), data...)
		dataErr = fmt.Errorf("Invalid JSON: %w", err)
	} else {
		data = formatted
		dataErr = nil
	}
	p.TextPreviewer.Preview(entry, data, dataErr, queueUpdateDraw)
}

const jsonIndent = "  "

func prettyJSON(input []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, input, "", jsonIndent)
	if err != nil {
		return input, err
	}
	return out.Bytes(), nil
}
