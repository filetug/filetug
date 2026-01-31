package profiling

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestDoCPUProfiling(t *testing.T) {
	t.Parallel()
	origOsCreate := osCreate
	defer func() { osCreate = origOsCreate }()

	t.Run("success", func(t *testing.T) {
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
	})

	t.Run("error_osCreate", func(t *testing.T) {
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
	})

	t.Run("error_pprofStartCPUProfile", func(t *testing.T) {
		tempFile := "cpu_err.prof"
		defer func() {
			_ = os.Remove(tempFile)
		}()

		osCreate = os.Create
		origStart := pprofStartCPUProfile
		defer func() { pprofStartCPUProfile = origStart }()

		pprofStartCPUProfile = func(w io.Writer) error {
			return errors.New("mock pprof error")
		}

		closeFunc := DoCPUProfiling(tempFile)
		if closeFunc == nil {
			t.Fatal("expected closeFunc to be not nil")
		}
		closeFunc()
	})
}
