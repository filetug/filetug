package filetug

import (
	"os"
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewDirSummary(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newDirSummary(nav)
	assert.NotNil(t, ds)
	assert.NotNil(t, ds.extTable)
}

func TestDirSummary_SetDir(t *testing.T) {
	app := tview.NewApplication()
	nav := NewNavigator(app)
	ds := newDirSummary(nav)

	entries := []os.DirEntry{
		mockDirEntry{name: "image1.png", isDir: false},
		mockDirEntry{name: "image2.png", isDir: false},
		mockDirEntry{name: "script.go", isDir: false},
		mockDirEntry{name: "unknown.foo", isDir: false},
		mockDirEntry{name: "subdir", isDir: true},
	}

	dir := &DirContext{
		Path:     "/test",
		children: entries,
	}

	ds.SetDir(dir)

	// .png -> Image, .go -> Code, .foo -> Other
	assert.Len(t, ds.extGroups, 3)

	var imageGroup *extensionsGroup
	for _, g := range ds.extGroups {
		if g.id == "Image" {
			imageGroup = g
			break
		}
	}
	if imageGroup == nil {
		t.Fatal("expected imageGroup to be not nil")
	}
	assert.Equal(t, "Images", imageGroup.title)
	assert.Len(t, imageGroup.extStats, 1) // .png
}

func TestGetSizeCell(t *testing.T) {
	cell := getSizeCell(1024, 0)
	assert.Equal(t, "  1KB", cell.Text)

	cell = getSizeCell(1024*1024*1024*1024, 0)
	assert.Contains(t, cell.Text, "1TB")
}
