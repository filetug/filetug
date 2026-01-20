package ftstate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentDir(t *testing.T) {
	// Setup temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "filetug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	origSettingsDirPath := settingsDirPath
	settingsDirPath = tmpDir
	defer func() { settingsDirPath = origSettingsDirPath }()

	origReadJSON := readJSON
	defer func() { readJSON = origReadJSON }()

	t.Run("empty_state", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return nil
		}
		dir := GetCurrentDir()
		if dir != "" {
			t.Errorf("expected empty dir, got %s", dir)
		}
	})

	t.Run("with_state", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			s := o.(*State)
			s.CurrentDir = "/some/dir"
			return nil
		}
		dir := GetCurrentDir()
		if dir != "/some/dir" {
			t.Errorf("expected /some/dir, got %s", dir)
		}
	})
}

func TestSaveCurrentDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filetug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	origSettingsDirPath := settingsDirPath
	settingsDirPath = tmpDir
	defer func() { settingsDirPath = origSettingsDirPath }()

	origReadJSON := readJSON
	origWriteJSON := writeJSON
	origLogErr := logErr
	defer func() {
		readJSON = origReadJSON
		writeJSON = origWriteJSON
		logErr = origLogErr
	}()

	t.Run("success", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return nil
		}
		writeJSON = func(filePath string, o interface{}) error {
			s := o.(State)
			if s.CurrentDir != "/new/dir" {
				t.Errorf("expected /new/dir, got %s", s.CurrentDir)
			}
			return nil
		}
		SaveCurrentDir("file://", "/new/dir")
	})

	t.Run("read_error", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return errors.New("read error")
		}
		// Should return early without calling writeJSON
		writeJSON = func(filePath string, o interface{}) error {
			return nil
		}
		SaveCurrentDir("file://", "/new/dir")
	})

	t.Run("mkdir_error", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return nil
		}

		// Use a path that cannot be created
		oldDirPath := settingsDirPath
		settingsDirPath = "/root/noaccess/dir"
		defer func() { settingsDirPath = oldDirPath }()

		SaveCurrentDir("fs", "/new/dir")
		// TODO: Assert error has been logged
	})

	t.Run("not_a_directory", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return nil
		}

		file, _ := os.CreateTemp(tmpDir, "notadir")
		_ = file.Close()

		oldDirPath := settingsDirPath
		settingsDirPath = file.Name()
		defer func() { settingsDirPath = oldDirPath }()

		var logCalled bool
		logErr = func(v ...interface{}) {
			logCalled = true
		}

		SaveCurrentDir("file://", "/new/dir")
		if !logCalled {
			t.Error("expected logErr to be called when settingsDirPath is a file")
		}
	})

	t.Run("write_error", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return nil
		}

		oldDirPath := settingsDirPath
		settingsDirPath = tmpDir
		defer func() { settingsDirPath = oldDirPath }()

		writeJSON = func(filePath string, o interface{}) error {
			return errors.New("write error")
		}

		var logCalled bool
		logErr = func(v ...interface{}) {
			logCalled = true
		}

		SaveCurrentDir("file://", "/new/dir")
		if !logCalled {
			t.Error("expected logErr to be called for writeJSON failure")
		}
	})
}

func TestGetState(t *testing.T) {
	origReadJSON := readJSON
	defer func() { readJSON = origReadJSON }()

	t.Run("success", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			s := o.(*State)
			s.CurrentDir = "/test/dir"
			return nil
		}
		state, err := GetState()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if state.CurrentDir != "/test/dir" {
			t.Errorf("expected /test/dir, got %s", state.CurrentDir)
		}
	})

	t.Run("error", func(t *testing.T) {
		readJSON = func(filePath string, required bool, o interface{}) error {
			return errors.New("read error")
		}
		_, err := GetState()
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestSaveSelectedTreeDir(t *testing.T) {
	origReadJSON := readJSON
	origWriteJSON := writeJSON
	defer func() {
		readJSON = origReadJSON
		writeJSON = origWriteJSON
	}()

	readJSON = func(filePath string, required bool, o interface{}) error {
		return nil
	}

	var savedState State
	writeJSON = func(filePath string, o interface{}) error {
		savedState = o.(State)
		return nil
	}

	SaveSelectedTreeDir("/some/path/to/dir/")
	// path.Split("/a/b/c/") -> "/a/b/c/", ""
	// In state.go: name, _ := path.Split(dir) -> state.SelectedTreeDir = name

	if savedState.SelectedTreeDir != "/some/path/to/dir/" {
		t.Errorf("expected /some/path/to/dir/, got %s", savedState.SelectedTreeDir)
	}
}

func TestSaveCurrentFileName(t *testing.T) {
	origReadJSON := readJSON
	origWriteJSON := writeJSON
	defer func() {
		readJSON = origReadJSON
		writeJSON = origWriteJSON
	}()

	readJSON = func(filePath string, required bool, o interface{}) error {
		return nil
	}

	var savedState State
	writeJSON = func(filePath string, o interface{}) error {
		savedState = o.(State)
		return nil
	}

	SaveCurrentFileName("file.txt")
	if savedState.CurrentDirEntry != "file.txt" {
		t.Errorf("expected file.txt, got %s", savedState.CurrentDirEntry)
	}
}

func TestSaveCurrentDir_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid URL")
		}
	}()
	SaveCurrentDir(":", "/dir")
}

func TestGetStateFilePath(t *testing.T) {
	oldDirPath := settingsDirPath
	settingsDirPath = "/tmp/test"
	defer func() { settingsDirPath = oldDirPath }()

	path := getStateFilePath()
	expected := filepath.Join("/tmp/test", stateFileName)
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
