package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rivo/tview"
)

func TestMainRoot(t *testing.T) {
	runCalled := false

	oldRun := run
	defer func() {
		run = oldRun
	}()
	run = func(app application) {
		runCalled = true
	}

	main()

	if !runCalled {
		t.Fatal("expected main function to call run")
	}
}

func Test_newApp(t *testing.T) {
	oldSetupApp := setupApp
	defer func() {
		setupApp = oldSetupApp
	}()
	setupAppCalled := false
	setupApp = func(app *tview.Application) {
		setupAppCalled = true
	}

	app := newApp()
	if app == nil {
		t.Errorf("newApp returned nil")
	}
	if !setupAppCalled {
		t.Errorf("expected newApp to call setupApp")
	}
}

type fakeApp struct {
	err error
}

func (f fakeApp) Run() error {
	return fmt.Errorf("app failed: %w", f.err)
}

func Test_run(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	var expectedErr = errors.New("test error")
	run(fakeApp{err: expectedErr})

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, expectedErr.Error()) {
		t.Errorf("expected stderr to contain %q, got %q", expectedErr.Error(), output)
	}
}

func Test_newFileTugApp(t *testing.T) {
	oldNewApp := newApp
	defer func() {
		newApp = oldNewApp
	}()
	newApp = func() *tview.Application {
		return tview.NewApplication()
	}

	t.Run("default", func(t *testing.T) {
		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
	})

	t.Run("with_pprof", func(t *testing.T) {
		*pprofAddr = "localhost:0" // Use port 0 for random available port
		defer func() { *pprofAddr = "" }()
		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
	})

	t.Run("with_cpuprofile", func(t *testing.T) {
		*cpuprofile = "cpuprofile"
		defer func() { *cpuprofile = "" }()

		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
	})

	t.Run("with_memprofile", func(t *testing.T) {
		*memprofile = "memprofile"
		defer func() { *memprofile = "" }()

		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
	})
}
