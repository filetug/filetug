package profiling

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestDoCPUProfiling(t *testing.T) {
	// Note: Cannot run with t.Parallel() due to global variable modifications
	origOsCreate := osCreate
	defer func() {
		osCreate = origOsCreate
	}()

	// Test success case
	tempFile := "cpu.prof"
	defer func() {
		_ = os.Remove(tempFile)
	}()

	osCreate = os.Create
	closeFunc := DoCPUProfiling(tempFile)
	if closeFunc == nil {
		t.Fatal("expected closeFunc to be not nil")
	}
	closeFunc()

	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Errorf("expected profile file to be created")
	}
}

func TestDoCPUProfiling_ErrorOsCreate(t *testing.T) {
	origOsCreate := osCreate
	defer func() {
		osCreate = origOsCreate
	}()
	
	t.Cleanup(func() {
		_ = os.Remove("invalid")
	})
	osCreate = func(name string) (*os.File, error) {
		return nil, errors.New("mock error")
	}
	closeFunc := DoCPUProfiling("invalid")
	if closeFunc == nil {
		t.Fatal("expected closeFunc to be not nil even on error (returns empty func)")
	}
	closeFunc()
}

func TestDoCPUProfiling_ErrorPprofStartCPUProfile(t *testing.T) {
	origOsCreate := osCreate
	origStart := pprofStartCPUProfile
	defer func() {
		osCreate = origOsCreate
		pprofStartCPUProfile = origStart
	}()
	
	tempFile := "cpu_err.prof"
	defer func() {
		_ = os.Remove(tempFile)
	}()

	osCreate = os.Create
	pprofStartCPUProfile = func(w io.Writer) error {
		return errors.New("mock pprof error")
	}

	closeFunc := DoCPUProfiling(tempFile)
	if closeFunc == nil {
		t.Fatal("expected closeFunc to be not nil")
	}
	closeFunc()
}
