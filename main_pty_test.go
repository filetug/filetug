//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"
)

const ptySmokeTimeout = 5 * time.Second

func TestCompiledBinaryPTYHappyPath(t *testing.T) {
	fixtureRoot := t.TempDir()
	docsPath := filepath.Join(fixtureRoot, "docs")
	if err := os.Mkdir(docsPath, 0o700); err != nil {
		t.Fatalf("create docs fixture: %v", err)
	}
	alphaPath := filepath.Join(docsPath, "01-alpha.txt")
	if err := os.WriteFile(alphaPath, []byte("ALPHA_PTY_MARKER\n"), 0o600); err != nil {
		t.Fatalf("write alpha fixture: %v", err)
	}
	targetPath := filepath.Join(docsPath, "02-target.md")
	if err := os.WriteFile(targetPath, []byte("# TARGET_PTY_MARKER\n\nPTY navigation reached the Markdown preview.\n"), 0o600); err != nil {
		t.Fatalf("write target fixture: %v", err)
	}

	binaryPath := filepath.Join(t.TempDir(), "filetug")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = ptySmokeWorkingDirectory(t)
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build root executable: %v\n%s", err, buildOutput)
	}

	command := exec.Command(binaryPath, fixtureRoot)
	command.Dir = fixtureRoot
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	initialSize := &pty.Winsize{Rows: 24, Cols: 80}
	terminal, err := pty.StartWithSize(command, initialSize)
	if err != nil {
		t.Fatalf("start root executable in PTY: %v", err)
	}
	rows, columns, err := pty.Getsize(terminal)
	if err != nil {
		t.Fatalf("read initial PTY dimensions: %v", err)
	}
	if rows != 24 || columns != 80 {
		t.Fatalf("initial PTY dimensions = %dx%d, want 24x80", rows, columns)
	}
	transcript := newPTYSmokeTranscript()
	readerDone := make(chan error, 1)
	go readPTYSmokeTranscript(terminal, transcript, readerDone)
	processDone := make(chan error, 1)
	go func() {
		processDone <- command.Wait()
	}()
	processWaited := false
	readerWaited := false
	t.Cleanup(func() {
		if !processWaited {
			_ = command.Process.Kill()
			timer := time.NewTimer(ptySmokeTimeout)
			select {
			case <-processDone:
			case <-timer.C:
				t.Errorf("root executable did not stop during cleanup")
			}
			timer.Stop()
		}
		if !readerWaited {
			_ = terminal.Close()
			timer := time.NewTimer(ptySmokeTimeout)
			select {
			case <-readerDone:
			case <-timer.C:
				t.Errorf("PTY reader did not stop during cleanup")
			}
			timer.Stop()
		}
	})

	transcript.waitForAll(t, 0, ptySmokeTimeout, "Directories", "Files", "Preview", "docs")
	if _, err := io.WriteString(terminal, "\x1b[B\r"); err != nil {
		t.Fatalf("open docs from directory tree: %v", err)
	}
	transcript.waitForAll(t, 0, ptySmokeTimeout, "01-alpha.txt", "02-target.md")
	if _, err := io.WriteString(terminal, "\t\x1b[B"); err != nil {
		t.Fatalf("focus files and select target: %v", err)
	}
	transcript.waitForAll(t, 0, ptySmokeTimeout, "TARGET_PTY_MARKER")
	focusOffset := transcript.offset()
	if _, err := io.WriteString(terminal, "\t"); err != nil {
		t.Fatalf("focus preview: %v", err)
	}
	transcript.waitForAll(t, focusOffset, ptySmokeTimeout, "TARGET_PTY_MARKER")

	resizeOffset := transcript.offset()
	resized := &pty.Winsize{Rows: 30, Cols: 100}
	if err := pty.Setsize(terminal, resized); err != nil {
		t.Fatalf("resize PTY: %v", err)
	}
	rows, columns, err = pty.Getsize(terminal)
	if err != nil {
		t.Fatalf("read resized PTY dimensions: %v", err)
	}
	if rows != 30 || columns != 100 {
		t.Fatalf("resized PTY dimensions = %dx%d, want 30x100", rows, columns)
	}
	transcript.waitForAll(t, resizeOffset, ptySmokeTimeout, "TARGET_PTY_MARKER")

	if _, err := io.WriteString(terminal, "q"); err != nil {
		t.Fatalf("quit root executable: %v", err)
	}
	processErr := awaitPTYSmokeProcess(t, processDone, transcript, "normal quit")
	processWaited = true
	if processErr != nil {
		t.Fatalf("root executable exit: %v", processErr)
	}
	readerErr := awaitPTYSmokeReader(t, terminal, readerDone, transcript, "normal quit")
	readerWaited = true
	if readerErr != nil {
		t.Fatalf("PTY reader: %v", readerErr)
	}

	raw := transcript.snapshot()
	assertPTYSmokeSequence(t, raw, ansi.SetModeAltScreenSaveCursor, ansi.ResetModeAltScreenSaveCursor, "alternate screen")
	assertPTYSmokeSequence(t, raw, ansi.HideCursor, ansi.ShowCursor, "cursor visibility")
}

type ptySmokeTranscript struct {
	mu      sync.Mutex
	raw     bytes.Buffer
	updated chan struct{}
}

func newPTYSmokeTranscript() *ptySmokeTranscript {
	return &ptySmokeTranscript{updated: make(chan struct{}, 1)}
}

func (t *ptySmokeTranscript) append(chunk []byte) {
	t.mu.Lock()
	_, _ = t.raw.Write(chunk)
	t.mu.Unlock()
	select {
	case t.updated <- struct{}{}:
	default:
	}
}

func (t *ptySmokeTranscript) offset() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.raw.Len()
}

func (t *ptySmokeTranscript) snapshot() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.raw.String()
}

func (t *ptySmokeTranscript) waitForAll(testingT *testing.T, offset int, timeout time.Duration, markers ...string) {
	testingT.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		raw := t.snapshot()
		if offset <= len(raw) {
			section := raw[offset:]
			stripped := ansi.Strip(section)
			found := true
			for _, marker := range markers {
				if !strings.Contains(stripped, marker) {
					found = false
					break
				}
			}
			if found {
				return
			}
		}
		select {
		case <-t.updated:
		case <-timer.C:
			raw = t.snapshot()
			stripped := ansi.Strip(raw)
			testingT.Fatalf("PTY transcript after offset %d did not contain %q within %s\nraw=%q\nstripped=%q", offset, markers, timeout, raw, stripped)
		}
	}
}

func readPTYSmokeTranscript(terminal *os.File, transcript *ptySmokeTranscript, done chan<- error) {
	buffer := make([]byte, 4096)
	for {
		count, err := terminal.Read(buffer)
		if count > 0 {
			transcript.append(buffer[:count])
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) || errors.Is(err, syscall.EIO) || errors.Is(err, os.ErrClosed) {
			done <- nil
			return
		}
		done <- err
		return
	}
}

func awaitPTYSmokeProcess(testingT *testing.T, done <-chan error, transcript *ptySmokeTranscript, phase string) error {
	testingT.Helper()
	timer := time.NewTimer(ptySmokeTimeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		raw := transcript.snapshot()
		stripped := ansi.Strip(raw)
		testingT.Fatalf("root executable did not exit during %s within %s\nraw=%q\nstripped=%q", phase, ptySmokeTimeout, raw, stripped)
		return nil
	}
}

func awaitPTYSmokeReader(testingT *testing.T, terminal *os.File, done <-chan error, transcript *ptySmokeTranscript, phase string) error {
	testingT.Helper()
	timer := time.NewTimer(ptySmokeTimeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		if err := terminal.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			testingT.Fatalf("close PTY after reader timeout: %v", err)
		}
		closer := time.NewTimer(ptySmokeTimeout)
		defer closer.Stop()
		select {
		case err := <-done:
			return err
		case <-closer.C:
			raw := transcript.snapshot()
			stripped := ansi.Strip(raw)
			testingT.Fatalf("PTY reader did not stop during %s within %s\nraw=%q\nstripped=%q", phase, ptySmokeTimeout, raw, stripped)
			return nil
		}
	}
}

func assertPTYSmokeSequence(testingT *testing.T, raw, first, second, name string) {
	testingT.Helper()
	firstIndex := strings.Index(raw, first)
	secondIndex := strings.LastIndex(raw, second)
	if firstIndex < 0 || secondIndex <= firstIndex {
		testingT.Fatalf("PTY %s sequence is missing or unordered: first=%q second=%q raw=%q", name, first, second, raw)
	}
}

func ptySmokeWorkingDirectory(testingT *testing.T) string {
	testingT.Helper()
	directory, err := os.Getwd()
	if err != nil {
		testingT.Fatalf("get test working directory: %v", err)
	}
	return directory
}
