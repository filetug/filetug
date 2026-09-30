package tui

import (
	"cmp"
	"context"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/filetug/filetug/pkg/files"
)

// fileExtTypes maps a file extension to the kind of file it names.
var fileExtTypes = map[string]string{
	".jpg": "Image", ".jpeg": "Image", ".png": "Image", ".gif": "Image", ".bmp": "Image",
	".riff": "Image", ".tiff": "Image", ".vp8": "Image", ".vp8l": "Image", ".webp": "Image",
	".mov": "Video", ".mp4": "Video", ".webm": "Video",
	".go": "Code", ".css": "Code", ".js": "Code", ".cpp": "Code", ".java": "Code", ".cs": "Code",
	".json": "Data", ".xml": "Data", ".dbf": "Data",
	".txt": "Text", ".md": "Text",
	".log": "Log",
}

// otherGroupID is the group of extensions that have no kind.
const otherGroupID = "Other"

// fileExtPlurals are the titles of the kinds whose plural is not the kind and an
// "s".
var fileExtPlurals = map[string]string{"Data": "Data", "Code": "Code"}

// extStat is how many files have an extension and how large they are together.
type extStat struct {
	ID    string
	Count int
	Size  int64
}

// extGroup is the extensions of one kind of file.
type extGroup struct {
	ID    string
	Title string
	Count int
	Size  int64
	Exts  []*extStat
}

// Extensions are the extensions of the group.
func (g *extGroup) Extensions() []string {
	exts := make([]string, len(g.Exts))
	for i, e := range g.Exts {
		exts[i] = e.ID
	}
	return exts
}

// dirSummary describes the content of a directory for the preview.
type dirSummary struct {
	Path   string
	Groups []*extGroup
	// Git is the status of the repository the directory is in; nil outside one.
	Git *gitDirStatus
}

// summarize groups the files of a directory by extension and kind, with the
// number and total size of the files of each. Directories and dot-files are not
// counted.
func summarize(entries []os.DirEntry) []*extGroup {
	byExt := map[string]*extStat{}
	byGroup := map[string]*extGroup{}
	var groups []*extGroup
	for _, entry := range entries {
		name := entry.Name()
		ext := path.Ext(name)
		if entry.IsDir() || ext == name {
			continue
		}
		stat, ok := byExt[ext]
		if !ok {
			stat = &extStat{ID: ext}
			byExt[ext] = stat
		}
		size := fileSize(entry)
		stat.Count++
		stat.Size += size

		kind := cmp.Or(fileExtTypes[ext], otherGroupID)
		group, ok := byGroup[kind]
		if !ok {
			group = &extGroup{ID: kind, Title: cmp.Or(fileExtPlurals[kind], kind+"s")}
			byGroup[kind] = group
			groups = append(groups, group)
		}
		group.Count++
		group.Size += size
		if !slices.Contains(group.Exts, stat) {
			group.Exts = append(group.Exts, stat)
		}
	}
	slices.SortFunc(groups, func(a, b *extGroup) int {
		switch {
		case a.ID == otherGroupID:
			return 1
		case b.ID == otherGroupID:
			return -1
		}
		return strings.Compare(a.Title, b.Title)
	})
	for _, g := range groups {
		slices.SortFunc(g.Exts, func(a, b *extStat) int { return strings.Compare(a.ID, b.ID) })
	}
	return groups
}

// fileSize is the size of an entry, 0 when it cannot be read.
func fileSize(entry os.DirEntry) int64 {
	info, err := entry.Info()
	if err != nil || isNil(info) {
		return 0
	}
	return info.Size()
}

// buildSummary reads a directory and describes its content. On the local file
// system the git status of the directory is included.
func buildSummary(store files.Store, dirPath string) (*dirSummary, error) {
	s := &dirSummary{Path: dirPath}
	if store == nil {
		return s, nil
	}
	entries, err := store.ReadDir(context.Background(), dirPath)
	if err != nil {
		return nil, err
	}
	s.Groups = summarize(entries)
	if store.RootURL().Scheme == "file" {
		s.Git = loadGitDirStatus(dirPath)
	}
	return s, nil
}
