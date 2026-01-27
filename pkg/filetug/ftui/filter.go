package ftui

import (
	"os"
	"path"
	"strings"
)

type FilterFunc func(os.DirEntry) bool

type Filter struct {
	ShowHidden bool
	ShowDirs   bool
	Extensions []string
	MaskFilter FilterFunc
}

func (f Filter) IsEmpty() bool {
	return len(f.Extensions) == 0
}

func (f Filter) IsVisible(entry os.DirEntry) (isVisible bool) {
	entryName := entry.Name()
	if !f.ShowHidden && strings.HasPrefix(entryName, ".") {
		return false
	}
	if entry.IsDir() && !f.ShowDirs {
		return false
	}
	if len(f.Extensions) > 0 {
		for _, ext := range f.Extensions {
			entryExt := path.Ext(entryName)
			if entryExt == ext {
				isVisible = true
				break
			}
		}
		if !isVisible {
			return false
		}
	}
	if f.MaskFilter != nil {
		if !f.MaskFilter(entry) {
			return false
		}
	}
	return true
}
