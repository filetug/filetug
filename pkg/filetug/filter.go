package filetug

import (
	"os"
	"path"
	"strings"
)

type Filter struct {
	ShowHidden bool
	ShowDirs   bool
	Extensions []string
}

func (f Filter) IsEmpty() bool {
	return len(f.Extensions) == 0
}

func (f Filter) IsVisible(entry os.DirEntry) bool {
	if !f.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
		return false
	}
	if entry.IsDir() && !f.ShowDirs {
		return false
	}
	if len(f.Extensions) == 0 {
		return true
	}
	for _, ext := range f.Extensions {
		if path.Ext(entry.Name()) == ext {
			return true
		}
	}
	return false
}
