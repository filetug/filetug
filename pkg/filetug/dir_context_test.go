package filetug

import (
	"os"
	"testing"

	"github.com/filetug/filetug/pkg/files"
)

// mockStoreForDirContext is a mock implementation of files.Store
type mockStoreForDirContext struct {
	files.Store
}

func (m *mockStoreForDirContext) RootTitle() string {
	return "Mock Store"
}

// mockDirEntryForDirContext is a mock implementation of os.DirEntry
type mockDirEntryForDirContext struct {
	name string
}

func (m mockDirEntryForDirContext) Name() string               { return m.name }
func (m mockDirEntryForDirContext) IsDir() bool                { return false }
func (m mockDirEntryForDirContext) Type() os.FileMode          { return 0 }
func (m mockDirEntryForDirContext) Info() (os.FileInfo, error) { return nil, nil }

func TestNewDirContext(t *testing.T) {
	store := &mockStoreForDirContext{}
	path := "/test/path"
	children := []os.DirEntry{
		mockDirEntryForDirContext{name: "file1"},
		mockDirEntryForDirContext{name: "file2"},
	}

	dc := newDirContext(store, path, children)

	if dc.Store != store {
		t.Errorf("Expected store %v, got %v", store, dc.Store)
	}
	if dc.Path != path {
		t.Errorf("Expected path %s, got %s", path, dc.Path)
	}
	if len(dc.children) != len(children) {
		t.Errorf("Expected %d children, got %d", len(children), len(dc.children))
	}
	for i, child := range children {
		if dc.children[i].Name() != child.Name() {
			t.Errorf("Expected child %d name %s, got %s", i, child.Name(), dc.children[i].Name())
		}
	}
}
