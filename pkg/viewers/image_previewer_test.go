package viewers

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/filetug/filetug/pkg/files"
)

func createImageFile(t *testing.T, dir, name string, width, height int, format string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 0, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create image file %s: %v", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	switch format {
	case "png":
		err = png.Encode(f, img)
	case "jpeg":
		err = jpeg.Encode(f, img, nil)
	default:
		t.Fatalf("unsupported format: %s", format)
	}
	if err != nil {
		t.Fatalf("failed to encode image %s: %v", path, err)
	}
	return path
}

func TestImagePreviewer_GetMeta(t *testing.T) {
	// Create a temporary directory for test images
	tmpDir, err := os.MkdirTemp("", "imageviewer_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	pngPath := createImageFile(t, tmpDir, "test.png", 100, 50, "png")
	jpegPath := createImageFile(t, tmpDir, "test.jpg", 200, 150, "jpeg")

	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("not an image"), 0644); err != nil {
		t.Fatalf("failed to create text file: %v", err)
	}

	previewer := ImagePreviewer{}

	t.Run("PNG image", func(t *testing.T) {
		meta := previewer.GetMeta(pngPath)
		if meta == nil {
			t.Fatal("expected meta, got nil")
		}
		if len(meta.Groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(meta.Groups))
		}
		group := meta.Groups[0]
		if group.Title != "Format: PNG" {
			t.Errorf("expected format PNG, got %s", group.Title)
		}

		var width, height string
		for _, r := range group.Records {
			switch r.ID {
			case "width":
				width = r.Value
			case "height":
				height = r.Value
			}
		}
		if width != "100" {
			t.Errorf("expected width 100, got %s", width)
		}
		if height != "50" {
			t.Errorf("expected height 50, got %s", height)
		}
	})

	t.Run("JPEG image", func(t *testing.T) {
		meta := previewer.GetMeta(jpegPath)
		if meta == nil {
			t.Fatal("expected meta, got nil")
		}
		group := meta.Groups[0]
		if group.Title != "Format: JPEG" {
			t.Errorf("expected format JPEG, got %s", group.Title)
		}

		var width, height string
		for _, r := range group.Records {
			switch r.ID {
			case "width":
				width = r.Value
			case "height":
				height = r.Value
			}
		}
		if width != "200" {
			t.Errorf("expected width 200, got %s", width)
		}
		if height != "150" {
			t.Errorf("expected height 150, got %s", height)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		meta := previewer.GetMeta(filepath.Join(tmpDir, "noexist.png"))
		if meta != nil {
			t.Errorf("expected nil meta for non-existent file, got %v", meta)
		}
	})

	t.Run("Invalid image file", func(t *testing.T) {
		meta := previewer.GetMeta(txtPath)
		if meta != nil {
			t.Errorf("expected nil meta for text file, got %v", meta)
		}
	})
}

func TestImagePreviewerNewAndPreview(t *testing.T) {
	previewer := NewImagePreviewer()
	meta := previewer.Meta()
	main := previewer.Main()
	if meta != nil {
		t.Errorf("expected nil meta, got %v", meta)
	}
	if main != previewer.metaTable {
		t.Errorf("expected main to be meta table")
	}

	tmpDir := t.TempDir()
	path := createImageFile(t, tmpDir, "preview.png", 20, 10, "png")
	_ = path
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "preview.png"}, tmpDir)

	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}

	previewer.PreviewSingle(entry, nil, nil, queueUpdateDraw)
	waitForUpdate(t, done)

	rowCount := previewer.metaTable.GetRowCount()
	if rowCount != 3 {
		t.Errorf("expected 3 rows, got %d", rowCount)
	}
}
