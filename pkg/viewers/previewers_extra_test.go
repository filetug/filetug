package viewers

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/filetug/filetug/pkg/files"
	"github.com/strongo/dsstore"
)

type errorLexer struct {
	config *chroma.Config
}

func (l *errorLexer) Config() *chroma.Config {
	return l.config
}

func (l *errorLexer) Tokenise(options *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	_, _ = options, text
	return nil, errors.New("tokenise failure")
}

func (l *errorLexer) SetRegistry(registry *chroma.LexerRegistry) chroma.Lexer {
	_ = registry
	return l
}

func (l *errorLexer) SetAnalyser(analyser func(text string) float32) chroma.Lexer {
	_ = analyser
	return l
}

func (l *errorLexer) AnalyseText(text string) float32 {
	_ = text
	return 1
}

func waitForUpdate(t *testing.T, done <-chan struct{}) {
	select {
	case <-done:
		return
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for preview update")
	}
}

func waitForText(t *testing.T, previewer *TextPreviewer, needle string) {
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for text %q", needle)
		case <-ticker.C:
			text := previewer.GetText(false)
			if strings.Contains(text, needle) {
				return
			}
		}
	}
}

func TestTextPreviewerPreviewPlainText(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewTextPreviewer(queueUpdateDraw)
	data := []byte("plain text")
	dir := filepath.Dir("note.unknownext")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	expected := string(data)
	assert.Equal(t, expected, text)
}

func TestTextPreviewerPreviewWithLexer(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewTextPreviewer(queueUpdateDraw)
	data := []byte("package main\n")
	dir := filepath.Dir("main.go")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "main.go"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "package")
}

func TestTextPreviewerPreviewWithLexerError(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}

	lexers.Register(&errorLexer{
		config: &chroma.Config{
			Name:      "ErrLexer",
			Filenames: []string{"*.errlex"},
		},
	})

	previewer := NewTextPreviewer(queueUpdateDraw)
	data := []byte("content")
	dir := filepath.Dir("file.errlex")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "file.errlex"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "Failed to format file")
}

func TestTextPreviewerPreviewWithLexerDataErr(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewTextPreviewer(queueUpdateDraw)
	data := []byte("package main\n")
	dir := filepath.Dir("main.go")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "main.go"}, dir)

	dataErr := errors.New("bad json")
	previewer.PreviewSingle(entry, data, dataErr)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "[red]bad json[-]")
	assert.Contains(t, text, "package")
}

func TestTextPreviewerPreviewPlainTextDataErr(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewTextPreviewer(queueUpdateDraw)
	data := []byte("plain text")
	dir := filepath.Dir("note.unknownext")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, dir)

	dataErr := errors.New("bad json")
	previewer.PreviewSingle(entry, data, dataErr)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "[red]bad json[-]")
	assert.Contains(t, text, "plain text")
}

func TestTextPreviewerPreviewReadsFile(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewTextPreviewer(queueUpdateDraw)

	content := []byte("file content")
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "note.unknownext")
	err := os.WriteFile(path, content, 0644)
	assert.NoError(t, err)

	entry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, tmpDir)

	previewer.PreviewSingle(entry, nil, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	expected := string(content)
	assert.Equal(t, expected, text)
}

func TestTextPreviewerPreviewReadFileError(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	queueUpdateDraw := func(fn func()) { fn() }
	previewer := NewTextPreviewer(queueUpdateDraw)
	tmpDir := t.TempDir()
	name := filepath.Base(tmpDir)
	dir := filepath.Dir(tmpDir)
	entry := files.NewEntryWithDirPath(mockDirEntry{name: name, isDir: true}, dir)

	previewer.PreviewSingle(entry, nil, nil)
	waitForText(t, previewer, "Failed to read file")
}

func TestTextPreviewerPreviewReadFileError_Stale(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	allow := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		<-allow
		fn()
	}
	previewer := NewTextPreviewer(queueUpdateDraw)
	dir := t.TempDir()
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "missing.txt"}, dir)

	previewer.PreviewSingle(entry, nil, nil)
	previewer.PreviewSingle(entry, []byte("fresh"), nil)
	close(allow)
}

func TestTextPreviewerPreviewQueueUpdateNil(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	previewer := NewTextPreviewer(nil)
	data := []byte("queue nil")
	dir := filepath.Dir("note.unknownext")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	// Should not panic and should not update text
	time.Sleep(100 * time.Millisecond)
	text := previewer.GetText(false)
	assert.Equal(t, "", text)
}

func TestTextPreviewerPreviewStalePlain(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	dir := filepath.Dir("note.unknownext")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, dir)

	allowFirst := make(chan struct{})
	doneFirst := make(chan struct{})
	doneSecond := make(chan struct{})

	queueUpdateFirst := func(fn func()) {
		<-allowFirst
		fn()
		select {
		case <-doneFirst:
		default:
			close(doneFirst)
		}
	}
	queueUpdateSecond := func(fn func()) {
		fn()
		select {
		case <-doneSecond:
		default:
			close(doneSecond)
		}
	}

	previewer := NewTextPreviewer(queueUpdateFirst)
	// First preview
	previewer.PreviewSingle(entry, []byte("first"), nil)
	// Second preview (reassigns queueUpdateDraw BEFORE first finishes)
	previewer.queueUpdateDraw = queueUpdateSecond
	previewer.PreviewSingle(entry, []byte("second"), nil)

	// Second should complete immediately (it uses queueUpdateSecond)
	waitForUpdate(t, doneSecond)

	// First is still blocked on allowFirst.
	// When released, it will call its p.queueUpdateDraw which is NOW queueUpdateSecond!
	// Wait, no. In the implementation of PreviewSingle:
	/*
		go func(previewID uint64) {
			if p.queueUpdateDraw == nil { return }
			...
			p.queueUpdateDraw(func() { ... })
		}(previewID)
	*/
	// It uses p.queueUpdateDraw BY REFERENCE at the time of call.
	// So first preview will call queueUpdateSecond if it's already reassigned.

	close(allowFirst)
	// We don't wait for doneFirst because it might not be called if p.isCurrentPreview fails
	// but p.isCurrentPreview check is INSIDE the func passed to p.queueUpdateDraw.
	// So p.queueUpdateDraw WILL be called.

	waitForText(t, previewer, "second")
}

func TestTextPreviewerPreviewStaleLexer(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	lexerDir := filepath.Dir("main.go")
	plainDir := filepath.Dir("note.unknownext")
	lexerEntry := files.NewEntryWithDirPath(mockDirEntry{name: "main.go"}, lexerDir)
	plainEntry := files.NewEntryWithDirPath(mockDirEntry{name: "note.unknownext"}, plainDir)

	allowFirst := make(chan struct{})
	doneFirst := make(chan struct{})
	doneSecond := make(chan struct{})

	queueUpdateFirst := func(fn func()) {
		<-allowFirst
		fn()
		select {
		case <-doneFirst:
		default:
			close(doneFirst)
		}
	}

	previewer := NewTextPreviewer(queueUpdateFirst)
	previewer.PreviewSingle(lexerEntry, []byte("package main\n"), nil)
	previewer.queueUpdateDraw = func(fn func()) {
		fn()
		select {
		case <-doneSecond:
		default:
			close(doneSecond)
		}
	}
	previewer.PreviewSingle(plainEntry, []byte("second"), nil)
	waitForUpdate(t, doneSecond)

	close(allowFirst)
	waitForText(t, previewer, "second")
}
func TestTextPreviewerMetaAndMain(t *testing.T) {
	t.Parallel()
	withTextPreviewerTestLock(t)
	previewer := NewTextPreviewer(nil)
	meta := previewer.Meta()
	main := previewer.Main()
	if meta != nil {
		t.Errorf("expected nil meta, got %v", meta)
	}
	if main != previewer.TextView {
		t.Errorf("expected main to be text view")
	}
}

func TestPrettyJSONSuccess(t *testing.T) {
	t.Parallel()
	input := []byte("{\"a\":1}")
	output, err := prettyJSON(input)
	assert.NoError(t, err)
	outputText := string(output)
	assert.Contains(t, outputText, "\n  \"a\": 1\n")
}

func TestJsonPreviewerPreviewReadsFile(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewJsonPreviewer(queueUpdateDraw)

	content := []byte("{\"a\":1}")
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.unknownext")
	err := os.WriteFile(path, content, 0644)
	assert.NoError(t, err)

	entry := files.NewEntryWithDirPath(mockDirEntry{name: "data.unknownext"}, tmpDir)

	previewer.PreviewSingle(entry, nil, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "\n  \"a\": 1\n")
}

func TestJsonPreviewerPreviewReadFileError(t *testing.T) {
	t.Parallel()
	previewer := NewJsonPreviewer(func(f func()) {})
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "missing.json"}, t.TempDir())
	previewer.PreviewSingle(entry, nil, nil)
}

func TestJsonPreviewerPreviewWithData(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewJsonPreviewer(queueUpdateDraw)
	data := []byte("{\"a\":1}")
	dir := filepath.Dir("data.unknownext")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "data.unknownext"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "\n  \"a\": 1\n")
}

func TestJsonPreviewerPreviewInvalidJSON(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewJsonPreviewer(queueUpdateDraw)
	data := []byte("{invalid}")
	dir := filepath.Dir("bad.json")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "bad.json"}, dir)

	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "[red]invalid JSON:")
	assert.Equal(t, 1, strings.Count(text, "invalid JSON:"))
}

func TestDsstorePreviewerPreviewSuccess(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewDsstorePreviewer(queueUpdateDraw)

	store := dsstore.Store{
		Records: []dsstore.Record{
			{
				FileName: "example",
				Type:     "bool",
				Data:     []byte{1},
				DataLen:  0,
			},
		},
	}
	var buffer bytes.Buffer
	err := store.Write(&buffer)
	assert.NoError(t, err)

	dir := filepath.Dir("test.DS_Store")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "test.DS_Store"}, dir)

	data := buffer.Bytes()
	previewer.PreviewSingle(entry, data, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "example: bool")
}

func TestDsstorePreviewerPreviewReadsFile(t *testing.T) {
	done := make(chan struct{})
	queueUpdateDraw := func(fn func()) {
		fn()
		close(done)
	}
	previewer := NewDsstorePreviewer(queueUpdateDraw)

	store := dsstore.Store{
		Records: []dsstore.Record{
			{
				FileName: "example",
				Type:     "bool",
				Data:     []byte{1},
				DataLen:  0,
			},
		},
	}
	var buffer bytes.Buffer
	err := store.Write(&buffer)
	assert.NoError(t, err)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "good.DS_Store")
	data := buffer.Bytes()
	err = os.WriteFile(path, data, 0644)
	assert.NoError(t, err)

	entry := files.NewEntryWithDirPath(mockDirEntry{name: "good.DS_Store"}, tmpDir)

	previewer.PreviewSingle(entry, nil, nil)
	waitForUpdate(t, done)

	text := previewer.GetText(false)
	assert.Contains(t, text, "example: bool")
}

func TestDsstorePreviewerPreviewReadFileError(t *testing.T) {
	t.Parallel()
	previewer := NewDsstorePreviewer(func(f func()) {})
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "missing.DS_Store"}, t.TempDir())
	previewer.PreviewSingle(entry, nil, nil)
}

func TestDsstorePreviewerPreviewError(t *testing.T) {
	t.Parallel()
	previewer := NewDsstorePreviewer(func(f func()) {})
	dir := filepath.Dir("bad.DS_Store")
	entry := files.NewEntryWithDirPath(mockDirEntry{name: "bad.DS_Store"}, dir)

	data := []byte("not a dsstore")
	previewer.PreviewSingle(entry, data, nil)

	text := previewer.GetText(false)
	assert.Contains(t, text, "Failed to read")
}
