package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/profiling"
	"github.com/filetug/filetug/pkg/tviewmocks"
	"go.uber.org/mock/gomock"
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
	setupApp = func(app navigator.App) {
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

type okApp struct{}

func (o okApp) Run() error {
	_ = o
	return nil
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

func Test_run_noError(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	run(okApp{})

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if output != "" {
		t.Errorf("expected no stderr output, got %q", output)
	}
}

func Test_newFileTugApp(t *testing.T) {
	oldNewApp := newApp
	defer func() {
		newApp = oldNewApp
	}()

	ctrl := gomock.NewController(t)

	newApp = func() navigator.App {
		return tviewmocks.NewMockApp(ctrl)
	}

	t.Run("default", func(t *testing.T) {
		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
	})

	t.Run("with_pprof", func(t *testing.T) {
		oldHTTPListenAndServe := httpListenAndServe
		oldPprofAddr := *pprofAddr
		defer func() {
			httpListenAndServe = oldHTTPListenAndServe
			*pprofAddr = oldPprofAddr
		}()
		
		serverStarted := make(chan struct{})
		// Mock the http server to synchronize with goroutine
		httpListenAndServe = func(addr string, handler http.Handler) error {
			close(serverStarted)
			return nil
		}
		
		*pprofAddr = "localhost:0" // Use port 0 for random available port
		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
		
		// Wait for goroutine to start to ensure it has read pprofAddr
		select {
		case <-serverStarted:
		case <-time.After(time.Second):
			t.Error("expected server to start")
		}
	})

	t.Run("with_cpuprofile", func(t *testing.T) {
		oldDoCPUProfiling := profiling.DoCPUProfiling
		defer func() {
			profiling.DoCPUProfiling = oldDoCPUProfiling
		}()
		closed := false
		called := false
		profiling.DoCPUProfiling = func(cpuProfFile string) func() {
			_ = cpuProfFile
			called = true
			return func() {
				closed = true
			}
		}

		*cpuProfile = "cpuprofile"
		defer func() { *cpuProfile = "" }()

		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
		if !called {
			t.Error("expected cpu profiling to be started")
		}
		if !closed {
			t.Error("expected cpu profiling to be stopped")
		}
	})

	t.Run("with_memprofile", func(t *testing.T) {
		oldDoMemProfiling := profiling.DoMemProfiling
		defer func() {
			profiling.DoMemProfiling = oldDoMemProfiling
		}()
		closed := false
		called := false
		profiling.DoMemProfiling = func(memProfFile string) func() {
			_ = memProfFile
			called = true
			return func() {
				closed = true
			}
		}

		*memProfile = "memprofile"
		defer func() { *memProfile = "" }()

		app := newFileTugApp()
		if app == nil {
			t.Error("newFileTugApp() returned nil")
		}
		if !called {
			t.Error("expected memory profiling to be started")
		}
		if !closed {
			t.Error("expected memory profiling to be stopped")
		}
	})
}

func Test_newFileTugApp_pprofError(t *testing.T) {
	oldNewApp := newApp
	oldListenAndServe := httpListenAndServe
	defer func() {
		newApp = oldNewApp
		httpListenAndServe = oldListenAndServe
	}()

	ctrl := gomock.NewController(t)

	newApp = func() navigator.App {
		return tviewmocks.NewMockApp(ctrl)
	}

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	listenCalled := make(chan struct{})
	var listenOnce sync.Once
	listenErr := errors.New("listen failed")
	httpListenAndServe = func(addr string, handler http.Handler) error {
		_ = addr
		_ = handler
		listenOnce.Do(func() {
			close(listenCalled)
		})
		return listenErr
	}

	*pprofAddr = "bad-address"
	defer func() { *pprofAddr = "" }()

	app := newFileTugApp()
	if app == nil {
		t.Error("newFileTugApp() returned nil")
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-listenCalled:
	case <-timer.C:
		t.Fatal("expected pprof server to start")
	}

	reader := bufio.NewReader(r)
	lineChan := make(chan string, 1)
	readErrChan := make(chan error, 1)
	go func() {
		line, err := reader.ReadString('\n')
		if err != nil {
			readErrChan <- err
			return
		}
		lineChan <- line
	}()

	timer = time.NewTimer(time.Second)
	defer timer.Stop()
	var output string
	select {
	case output = <-lineChan:
	case err := <-readErrChan:
		t.Fatalf("expected stderr output, got error: %v", err)
	case <-timer.C:
		t.Fatal("expected stderr output")
	}

	_ = w.Close()

	if !strings.Contains(output, "pprof server error") {
		t.Errorf("expected stderr to include pprof error, got %q", output)
	}
}

func Test_newFileTugApp_panicRecovery(t *testing.T) {
	oldNewApp := newApp
	oldExit := osExit
	oldStop := pprofStopCPUProfile
	defer func() {
		newApp = oldNewApp
		osExit = oldExit
		pprofStopCPUProfile = oldStop
	}()
	newApp = func() navigator.App {
		panic("boom")
	}

	exitCode := 0
	osExit = func(code int) {
		exitCode = code
	}
	stopCalled := false
	pprofStopCPUProfile = func() {
		stopCalled = true
	}

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	app := newFileTugApp()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if app != nil {
		t.Error("expected newFileTugApp() to return nil after panic")
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !stopCalled {
		t.Error("expected CPU profiling to stop on panic")
	}
	if !strings.Contains(output, "Recovered from panic") {
		t.Errorf("expected stderr to include panic recovery message, got %q", output)
	}
}
