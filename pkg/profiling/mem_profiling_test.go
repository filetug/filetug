package profiling

import (
	"errors"
	"io"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

func TestDoMemProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test with goroutines in short mode")
	}
	// Note: Cannot run with t.Parallel() due to spawned goroutines that never stop
	origOsCreate := osCreate
	origInterval := memProfilingInterval
	origPprofWrite := pprofWriteHeapProfile
	defer func() {
		osCreate = origOsCreate
		memProfilingInterval = origInterval
		pprofWriteHeapProfile = origPprofWrite
		// Wait longer for any goroutines to finish reading globals
		time.Sleep(500 * time.Millisecond)
	}()

	memProfilingInterval = 100 * time.Millisecond

	// Test success case
	tempFile := "mem.prof"
	defer func() {
		_ = os.Remove(tempFile)
	}()

	osCreate = os.Create
	pprofWriteHeapProfile = func(w io.Writer) error {
		return pprof.WriteHeapProfile(w)
	}
	
	writeMemProfile := DoMemProfiling(tempFile)
	if writeMemProfile == nil {
		t.Fatal("expected writeMemProfile to be not nil")
	}

	// Manually trigger once
	writeMemProfile()

	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Errorf("expected profile file to be created")
	}

	// Wait for goroutine to run at least once
	time.Sleep(300 * time.Millisecond)
}

func TestDoMemProfiling_ErrorOsCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test with goroutines in short mode")
	}
	origOsCreate := osCreate
	origInterval := memProfilingInterval
	origPprofWrite := pprofWriteHeapProfile
	defer func() {
		osCreate = origOsCreate
		memProfilingInterval = origInterval
		pprofWriteHeapProfile = origPprofWrite
		time.Sleep(500 * time.Millisecond)
	}()
	
	memProfilingInterval = 100 * time.Millisecond
	osCreate = func(name string) (*os.File, error) {
		return nil, errors.New("mock error")
	}
	pprofWriteHeapProfile = func(w io.Writer) error {
		return pprof.WriteHeapProfile(w)
	}
	
	t.Cleanup(func() {
		_ = os.Remove("invalid")
	})
	writeMemProfile := DoMemProfiling("invalid")
	writeMemProfile()
	time.Sleep(300 * time.Millisecond)
}

func TestDoMemProfiling_ErrorPprofWriteHeapProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test with goroutines in short mode")
	}
	origOsCreate := osCreate
	origInterval := memProfilingInterval
	origPprofWrite := pprofWriteHeapProfile
	defer func() {
		osCreate = origOsCreate
		memProfilingInterval = origInterval
		pprofWriteHeapProfile = origPprofWrite
		time.Sleep(500 * time.Millisecond)
	}()
	
	memProfilingInterval = 100 * time.Millisecond
	tempFile := "mem_err.prof"
	defer func() {
		_ = os.Remove(tempFile)
	}()

	osCreate = os.Create
	pprofWriteHeapProfile = func(w io.Writer) error {
		return errors.New("mock pprof error")
	}

	writeMemProfile := DoMemProfiling(tempFile)
	writeMemProfile()
	time.Sleep(300 * time.Millisecond)
}
