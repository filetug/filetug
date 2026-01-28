package filetug

import (
	"github.com/filetug/filetug/pkg/files"
	"github.com/rivo/tview"
)

type browser interface {
	GetCurrentEntry() files.EntryWithDirPath
	tview.Primitive
}

type current struct {
	dir string
	//entry os.DirEntry
}
