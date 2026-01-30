package viewers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/strongo/dsstore"
)

var _ Previewer = (*DsstorePreviewer)(nil)

type DsstorePreviewer struct {
	TextPreviewer
}

func NewDsstorePreviewer(queueUpdateDraw func(func())) *DsstorePreviewer {
	previewer := NewTextPreviewer(queueUpdateDraw)
	return &DsstorePreviewer{
		TextPreviewer: *previewer,
	}
}

func (p DsstorePreviewer) PreviewSingle(entry files.EntryWithDirPath, data []byte, dataErr error) {
	if data == nil {
		fullName := entry.FullName()
		var err error
		data, err = fsutils.ReadFileData(fullName, 0)
		if err != nil {
			return
		}
	}
	bufferRead := bytes.NewBuffer(data)
	var s dsstore.Store
	err := s.Read(bufferRead)
	if err != nil {
		errText := fmt.Sprintf("Failed to read %s: %s", entry.Name(), err.Error())
		p.showError(errText)
		return
	}
	var sb strings.Builder
	for _, r := range s.Records {
		_, _ = fmt.Fprintf(&sb, "%s: %s\n", r.FileName, r.Type)
	}
	content := sb.String()
	data = []byte(content)
	p.TextPreviewer.PreviewSingle(entry, data, dataErr)
}
