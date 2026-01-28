package viewers

import (
	"github.com/filetug/filetug/pkg/files"
	"github.com/rivo/tview"
)

type Previewer interface {
	Preview(entry files.EntryWithDirPath, data []byte, dataErr error, queueUpdateDraw func(func()))
	Meta() tview.Primitive
	Main() tview.Primitive
}

type Meta struct {
	Groups []*MetaGroup
}

type MetaGroup struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Records []*MetaRecord `json:"records"`
}

type MetaRecord struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Value string `json:"value"`
	//TitleAlign Align
	ValueAlign Align
}

type Align int

const (
	AlignLeft Align = iota
	AlignRight
)
