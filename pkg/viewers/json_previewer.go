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

func NewJsonPreviewer(queueUpdateDraw func(func())) *JsonPreviewer {
	textPreviewer := NewTextPreviewer(queueUpdateDraw)
	return &JsonPreviewer{
		TextPreviewer: *textPreviewer,
	}
}

func (p JsonPreviewer) PreviewSingle(entry files.EntryWithDirPath, data []byte, _ error) {
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
		prefix := "invalid JSON: " + errText + "\n"
		data = append([]byte(prefix), data...)
		err = fmt.Errorf("invalid JSON: %w", err)
	} else {
		data = formatted
		err = nil
	}
	p.TextPreviewer.PreviewSingle(entry, data, err)
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
