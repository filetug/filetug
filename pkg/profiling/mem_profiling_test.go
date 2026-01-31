package profiling

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestDoMemProfiling(t *testing.T) {
	t.Parallel()
	origOsCreate := osCreate
	origInterval := memProfilingInterval
	defer func() {
		osCreate = origOsCreate
		memProfilingInterval = origInterval
	}()

	memProfilingInterval = 100 * time.Millisecond

	t.Run("success", func(t *testing.T) {
		tempFile := "mem.prof"
		defer func() {
			_ = os.Remove(tempFile)
		}()

		osCreate = os.Create
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
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("error_osCreate", func(t *testing.T) {
		t.Cleanup(func() {
			_ = os.Remove("invalid")
		})
		osCreate = func(name string) (*os.File, error) {
			return nil, errors.New("mock error")
		}
		writeMemProfile := DoMemProfiling("invalid")
		writeMemProfile()
	})

	t.Run("error_pprofWriteHeapProfile", func(t *testing.T) {
		tempFile := "mem_err.prof"
		defer func() {
			_ = os.Remove(tempFile)
		}()

		osCreate = os.Create
		origWrite := pprofWriteHeapProfile
		defer func() { pprofWriteHeapProfile = origWrite }()

		pprofWriteHeapProfile = func(w io.Writer) error {
			return errors.New("mock pprof error")
		}

		writeMemProfile := DoMemProfiling(tempFile)
		writeMemProfile()
	})
}
