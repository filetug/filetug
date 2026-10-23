package tui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // register the decoders read by imageMeta
	_ "image/jpeg" // register the decoders read by imageMeta
	_ "image/png"  // register the decoders read by imageMeta
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/filetug/filetug/pkg/files"
	"github.com/filetug/filetug/pkg/fsutils"
	"github.com/strongo/dsstore"
	"github.com/tuigoff/tuigoff/pkg/highlight"
	"github.com/tuigoff/tuigoff/pkg/mdrender"
	_ "golang.org/x/image/bmp"  // register the decoders read by imageMeta
	_ "golang.org/x/image/riff" // register the decoders read by imageMeta
	_ "golang.org/x/image/vp8"  // register the decoders read by imageMeta
	_ "golang.org/x/image/vp8l" // register the decoders read by imageMeta
	_ "golang.org/x/image/webp" // register the decoders read by imageMeta
)

// textPreviewBytes is how much of a text file is previewed.
const textPreviewBytes = 10 * 1024

// previewMsg is the result of building a preview.
type previewMsg struct {
	Seq uint64
	// Title is the name of the previewed entry.
	Title string
	// Size and Modified are the entry's attributes; empty when unknown.
	Size, Modified string
	// Body is the styled text to show.
	Body string
	// Summary describes a directory; Body is then empty.
	Summary *dirSummary
	// Err is why the preview could not be built; Body is then empty.
	Err error
}

// Seams over the file system, replaced by tests.
var (
	readFileData = fsutils.ReadFileData
	openFile     = os.Open
	colorize     = highlight.Colorize
)

// previewCmd builds the preview of an entry off the event loop.
func previewCmd(seq uint64, store files.Store, entry files.EntryWithDirPath, isDir bool, width int) tea.Cmd {
	return func() tea.Msg { return buildPreview(seq, store, entry, isDir, width) }
}

// buildPreview reads an entry and renders it for a preview of the given width.
func buildPreview(seq uint64, store files.Store, entry files.EntryWithDirPath, isDir bool, width int) previewMsg {
	msg := previewMsg{Seq: seq, Title: entry.Name()}
	if msg.Title == "" {
		_, msg.Title = path.Split(entry.FullName())
	}
	if info, err := entry.Info(); err == nil && !isNil(info) {
		msg.Size = fsutils.GetSizeShortText(info.Size())
		msg.Modified = info.ModTime().Format(time.RFC3339)
	}
	if isDir {
		msg.Summary, msg.Err = buildSummary(store, entry.FullName())
		return msg
	}
	msg.Body, msg.Err = fileText(entry, width)
	return msg
}

// fileText renders the content of a file according to its type.
func fileText(entry files.EntryWithDirPath, width int) (string, error) {
	name, full := entry.Name(), entry.FullName()
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case name == ".DS_Store":
		return readAndRender(full, 0, dsstoreText)
	case ext == ".md":
		return readAndRender(full, 0, func(_ string, data []byte) (string, error) {
			return mdrender.Render(string(data), width), nil
		})
	case ext == ".json" || ext == ".jsonb":
		return readAndRender(full, 0, jsonText)
	case isImageExt(ext):
		return imageMeta(full)
	}
	return readAndRender(full, textPreviewBytes, highlightedText)
}

// readAndRender reads up to max bytes of a file (all when max is 0) and renders
// them.
func readAndRender(full string, max int, render func(name string, data []byte) (string, error)) (string, error) {
	data, err := readFileData(full, max)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file %s: %w", full, err)
	}
	return render(filepath.Base(full), data)
}

// highlightedText colours source text by the lexer its file name selects, and
// shows other text as it is.
func highlightedText(name string, data []byte) (string, error) {
	lexer := lexers.Match(name)
	if lexer == nil {
		return string(data), nil
	}
	text, err := colorize(string(data), highlight.DefaultStyle, lexer)
	if err != nil {
		return "", fmt.Errorf("failed to format file: %w", err)
	}
	return text, nil
}

// jsonText indents JSON and colours it; text that is not valid JSON is shown
// as it is under the reason.
func jsonText(name string, data []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		text, _ := highlightedText(name, data)
		return "invalid JSON: " + err.Error() + "\n" + text, nil
	}
	return highlightedText(name, out.Bytes())
}

// dsstoreText lists the records of a .DS_Store file.
func dsstoreText(name string, data []byte) (string, error) {
	var s dsstore.Store
	if err := s.Read(bytes.NewBuffer(data)); err != nil {
		return "", fmt.Errorf("failed to read %s: %w", name, err)
	}
	var sb strings.Builder
	for _, r := range s.Records {
		_, _ = fmt.Fprintf(&sb, "%s: %s\n", r.FileName, r.Type)
	}
	return sb.String(), nil
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".riff": true, ".tiff": true, ".vp8": true, ".webp": true,
}

func isImageExt(ext string) bool { return imageExts[ext] }

// imageMeta describes an image: its format and size in pixels.
func imageMeta(full string) (string, error) {
	f, err := openFile(full)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", full, err)
	}
	defer func() { _ = f.Close() }()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return "", fmt.Errorf("failed to read image %s: %w", full, err)
	}
	return fmt.Sprintf("Format: %s\n  Width  %d\n  Height %d\n", strings.ToUpper(format), cfg.Width, cfg.Height), nil
}
