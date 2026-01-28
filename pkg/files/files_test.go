package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirEntry(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		name := "testfile"
		const isDir = false
		de := NewDirEntry(name, isDir)

		if de.Name() != name {
			t.Errorf("expected Name() = %v, got %v", name, de.Name())
		}
		if de.IsDir() != isDir {
			t.Errorf("expected IsDir() = %v, got %v", isDir, de.IsDir())
		}
		if de.Type() != 0 {
			t.Errorf("expected Type() = 0, got %v", de.Type())
		}
		info, err := de.Info()
		if err != nil {
			t.Errorf("expected no error from Info(), got %v", err)
		}
		if info != nil {
			t.Errorf("expected nil info when no options provided, got %v", info)
		}
	})

	t.Run("directory", func(t *testing.T) {
		name := "testdir"
		const isDir = true
		de := NewDirEntry(name, isDir)

		if de.Name() != name {
			t.Errorf("expected Name() = %v, got %v", name, de.Name())
		}
		if de.IsDir() != isDir {
			t.Errorf("expected IsDir() = %v, got %v", isDir, de.IsDir())
		}
		if de.Type() != os.ModeDir {
			t.Errorf("expected Type() = %v, got %v", os.ModeDir, de.Type())
		}
	})

	t.Run("with_info", func(t *testing.T) {
		name := "testfile"
		const isDir = false
		size := int64(123)
		modTime := time.Now()
		de := NewDirEntry(name, isDir, Size(size), ModTime(modTime))

		info, err := de.Info()
		if err != nil {
			t.Errorf("expected no error from Info(), got %v", err)
		}
		if info == nil {
			t.Fatal("expected non-nil info when options provided")
		}
		if info.Name() != name {
			t.Errorf("expected info.Name() = %v, got %v", name, info.Name())
		}
		if info.Size() != size {
			t.Errorf("expected info.Size() = %v, got %v", size, info.Size())
		}
		if !info.ModTime().Equal(modTime) {
			t.Errorf("expected info.ModTime() = %v, got %v", modTime, info.ModTime())
		}
		if info.IsDir() != isDir {
			t.Errorf("expected info.IsDir() = %v, got %v", isDir, info.IsDir())
		}
		if info.Mode() != de.Type() {
			t.Errorf("expected info.Mode() = %v, got %v", de.Type(), info.Mode())
		}
		if info.Sys() != nil {
			t.Errorf("expected info.Sys() = nil, got %v", info.Sys())
		}
	})
}

type pathDirEntry struct {
	name string
}

func (p pathDirEntry) Name() string      { return p.name }
func (p pathDirEntry) IsDir() bool       { return false }
func (p pathDirEntry) Type() os.FileMode { return 0 }
func (p pathDirEntry) Info() (os.FileInfo, error) {
	return nil, nil
}

func TestNewDirEntry_PanicsOnNameWithPath(t *testing.T) {
	name := filepath.Join("parent", "child")
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for name with path")
		}
	}()
	_ = NewDirEntry(name, false)
}

func TestNewEntryWithDirPath_PanicsOnNameWithPath(t *testing.T) {
	entry := pathDirEntry{name: "parent/child"}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for entry name with path")
		}
	}()
	_ = NewEntryWithDirPath(entry, "/tmp")
}

func TestFileInfo_NilReceiver(t *testing.T) {
	var f *FileInfo
	if f.Name() != "" {
		t.Errorf("expected empty name for nil FileInfo")
	}
	if f.Size() != 0 {
		t.Errorf("expected 0 size for nil FileInfo")
	}
	if f.Mode() != 0 {
		t.Errorf("expected 0 mode for nil FileInfo")
	}
	if !f.ModTime().IsZero() {
		t.Errorf("expected zero modTime for nil FileInfo")
	}
	if f.IsDir() {
		t.Errorf("expected false for IsDir() for nil FileInfo")
	}
	if f.Sys() != nil {
		t.Errorf("expected nil for Sys() for nil FileInfo")
	}
}

func TestEntryWithDirPath(t *testing.T) {
	entry := NewDirEntry("test.txt", false)
	dir := "/home/user"
	e := NewEntryWithDirPath(entry, dir)

	if e.DirPath() != dir {
		t.Errorf("expected Dir = %v, got %v", dir, e.DirPath())
	}
	if e.Name() != "test.txt" {
		t.Errorf("expected Name() = %v, got %v", "test.txt", e.Name())
	}

	expectedPath := "/home/user/test.txt"
	if e.FullName() != expectedPath {
		t.Errorf("expected FullName() = %v, got %v", expectedPath, e.FullName())
	}

	expectedString := "/home/user/test.txt"
	if e.String() != expectedString {
		t.Errorf("expected String() = %v, got %v", expectedString, e.String())
	}
}
