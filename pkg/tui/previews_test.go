package tui

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/strongo/dsstore"
	"github.com/tuigoff/tuigoff/pkg/uitest"
)

func realEntry(t *testing.T, full string) files.EntryWithDirPath {
	t.Helper()
	de, err := os.ReadDir(filepath.Dir(full))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range de {
		if e.Name() == filepath.Base(full) {
			return files.NewEntryWithDirPath(e, filepath.Dir(full))
		}
	}
	t.Fatalf("%s not found", full)
	return nil
}

func preview(t *testing.T, full string) previewMsg {
	t.Helper()
	return buildPreview(3, osStore(), realEntry(t, full), false, 60)
}

func TestBuildPreviewText(t *testing.T) {
	dir := tree(t)
	msg := preview(t, filepath.Join(dir, "a.go"))
	if msg.Seq != 3 || msg.Title != "a.go" || msg.Size != "29B" || msg.Modified == "" || msg.Err != nil {
		t.Fatalf("%+v", msg)
	}
	if !strings.Contains(uitest.Plain(msg.Body), "package main") || msg.Body == uitest.Plain(msg.Body) {
		t.Fatalf("Go source is highlighted: %q", msg.Body)
	}
	write(t, filepath.Join(dir, "notes.unknown-ext"), "plain <text>")
	if got := preview(t, filepath.Join(dir, "notes.unknown-ext")).Body; got != "plain <text>" {
		t.Fatalf("text with no lexer is shown as it is: %q", got)
	}
}

func TestBuildPreviewMarkdownJSONAndDSStore(t *testing.T) {
	dir := tree(t)
	if got := uitest.Plain(preview(t, filepath.Join(dir, "README.md")).Body); !strings.Contains(got, "Title") || strings.Contains(got, "# Title") {
		t.Fatalf("Markdown is rendered: %q", got)
	}

	write(t, filepath.Join(dir, "d.json"), `{"a":1}`)
	if got := uitest.Plain(preview(t, filepath.Join(dir, "d.json")).Body); !strings.Contains(got, "  \"a\": 1") {
		t.Fatalf("JSON is indented: %q", got)
	}
	write(t, filepath.Join(dir, "bad.json"), `{"a":`)
	if got := uitest.Plain(preview(t, filepath.Join(dir, "bad.json")).Body); !strings.HasPrefix(got, "invalid JSON: ") || !strings.Contains(got, `{"a":`) {
		t.Fatalf("invalid JSON is shown under the reason: %q", got)
	}

	write(t, filepath.Join(dir, ".DS_Store"), "not a store")
	if msg := preview(t, filepath.Join(dir, ".DS_Store")); msg.Err == nil || !strings.Contains(msg.Err.Error(), "Failed to read .DS_Store") {
		t.Fatalf("a broken .DS_Store is reported: %+v", msg)
	}
}

func TestBuildPreviewDSStoreRecords(t *testing.T) {
	store := dsstore.Store{Records: []dsstore.Record{{FileName: "notes", Type: "long", Data: []byte{0, 0, 0, 1}}}}
	var buf bytes.Buffer
	if err := store.Write(&buf); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := preview(t, filepath.Join(dir, ".DS_Store")).Body; got != "notes: long\n" {
		t.Fatalf("records are listed as name: type: %q", got)
	}
}

func TestBuildPreviewImage(t *testing.T) {
	dir := tree(t)
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 7, 5))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := preview(t, filepath.Join(dir, "p.png")).Body; !strings.Contains(got, "Format: PNG") || !strings.Contains(got, "Width  7") || !strings.Contains(got, "Height 5") {
		t.Fatalf("image metadata: %q", got)
	}
	write(t, filepath.Join(dir, "broken.png"), "not an image")
	if msg := preview(t, filepath.Join(dir, "broken.png")); msg.Err == nil || !strings.Contains(msg.Err.Error(), "Failed to read image") {
		t.Fatalf("a broken image is reported: %+v", msg)
	}
	oldOpen := openFile
	openFile = func(string) (*os.File, error) { return nil, errors.New("denied") }
	defer func() { openFile = oldOpen }()
	if msg := preview(t, filepath.Join(dir, "p.png")); msg.Err == nil || !strings.Contains(msg.Err.Error(), "denied") {
		t.Fatalf("an unreadable image is reported: %+v", msg)
	}
}

func TestBuildPreviewErrors(t *testing.T) {
	dir := tree(t)
	oldRead := readFileData
	defer func() { readFileData = oldRead }()

	readFileData = func(string, int) ([]byte, error) { return nil, errors.New("denied") }
	if msg := preview(t, filepath.Join(dir, "a.go")); msg.Err == nil || !strings.Contains(msg.Err.Error(), "Failed to read file") {
		t.Fatalf("a read error is reported: %+v", msg)
	}
	readFileData = func(string, int) ([]byte, error) { return []byte("abc"), io.EOF }
	if msg := preview(t, filepath.Join(dir, "README.md")); msg.Err != nil || !strings.Contains(uitest.Plain(msg.Body), "abc") {
		t.Fatalf("EOF is not an error: %+v", msg)
	}
}

func TestHighlightedTextReportsALexerFailure(t *testing.T) {
	// A lexer that cannot tokenise makes the highlight fail; provoke it with
	// invalid UTF-8 for a lexer that rejects it is not portable, so the error
	// path is exercised through the seam instead.
	old := colorize
	colorize = func(string, string, chroma.Lexer) (string, error) { return "", errors.New("bad") }
	defer func() { colorize = old }()
	if _, err := highlightedText("x.go", []byte("package x")); err == nil || !strings.Contains(err.Error(), "Failed to format file") {
		t.Fatalf("got %v", err)
	}
}

func TestBuildPreviewDirectory(t *testing.T) {
	dir := tree(t)
	entry := realEntry(t, filepath.Join(dir, "alpha"))
	msg := buildPreview(1, osStore(), entry, true, 60)
	if msg.Body != "1 directories, 0 files" || msg.Err != nil {
		t.Fatalf("%+v", msg)
	}
	msg = buildPreview(1, nil, entry, true, 60)
	if msg.Body != "" || msg.Err != nil {
		t.Fatalf("no store, nothing to count: %+v", msg)
	}
	msg = buildPreview(1, fakeStore{root: url.URL{Scheme: "fake"}, err: errors.New("boom")}, entry, true, 60)
	if msg.Err == nil {
		t.Fatalf("a read error is reported: %+v", msg)
	}
}

func TestBuildPreviewTitleFallsBackToThePath(t *testing.T) {
	entry := files.NewEntryWithDirPath(files.NewDirEntry("", true), "/some/dir")
	if msg := buildPreview(1, nil, entry, true, 10); msg.Title != "dir" {
		t.Fatalf("title %q", msg.Title)
	}
}
