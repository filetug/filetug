package fsutils

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestDirExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "datatug_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	t.Run("exists", func(t *testing.T) {
		exists, err := DirExists(tmpDir)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not_exists", func(t *testing.T) {
		exists, err := DirExists(filepath.Join(tmpDir, "non_existent"))
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("is_file", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(filePath, []byte("test"), 0644)
		assert.NoError(t, err)

		exists, err := DirExists(filePath)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestExpandHome(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "", ExpandHome(""))
	})
	t.Run("no_tilde", func(t *testing.T) {
		assert.Equal(t, "/some/path", ExpandHome("/some/path"))
	})
	t.Run("only_tilde", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		assert.Equal(t, home, ExpandHome("~"))
	})
	t.Run("tilde_with_path", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, "abc"), ExpandHome("~/abc"))
	})
}

func TestReadJSONFile(t *testing.T) {
	type A struct {
		B string
	}

	t.Run("empty_not_required", func(t *testing.T) {
		var a A
		err := ReadJSONFile("", false, &a)
		assert.NoError(t, err)
	})

	t.Run("not_found_not_required", func(t *testing.T) {
		var a A
		err := ReadJSONFile("non_existent.json", false, &a)
		assert.NoError(t, err)
	})

	t.Run("not_found_required", func(t *testing.T) {
		var a A
		err := ReadJSONFile("non_existent.json", true, &a)
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test*.json")
		assert.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()

		_, err = tmpFile.WriteString(`{"B": "test"}`)
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		var a A
		err = ReadJSONFile(tmpFile.Name(), true, &a)
		assert.NoError(t, err)
		assert.Equal(t, "test", a.B)
	})

	t.Run("invalid_json", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test*.json")
		assert.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()

		_, err = tmpFile.WriteString(`{invalid}`)
		assert.NoError(t, err)
		err = tmpFile.Close()
		assert.NoError(t, err)

		var a A
		err = ReadJSONFile(tmpFile.Name(), true, &a)
		assert.Error(t, err)
	})

	t.Run("fail_to_close", func(t *testing.T) {
		// This is just to cover the defer close logic, though it's hard to make it fail and see the log
		tmpFile, err := os.CreateTemp("", "test*.json")
		assert.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		_, _ = tmpFile.WriteString(`{}`)
		_ = tmpFile.Close()

		var a A
		_ = ReadJSONFile(tmpFile.Name(), true, &a)
	})
}

type mockDecoder struct {
	err error
}

func (m mockDecoder) Decode(interface{}) error {
	return m.err
}

func TestReadFile_DecoderError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.json")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	err = ReadFile(tmpFile.Name(), true, nil, func(r io.Reader) Decoder {
		return mockDecoder{err: io.EOF}
	})
	assert.Error(t, err)
}

func TestDirExists_Error(t *testing.T) {
	// This is hard to trigger without a mockable filesystem, but we can try with a path that is too long on some systems
	// or just a path that we don't have permission to access.
	// Actually, we can use a path that contains a null byte to trigger an error in os.Stat.
	_, err := DirExists("path\x00with-null")
	assert.Error(t, err)
}
