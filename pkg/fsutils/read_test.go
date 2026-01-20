package fsutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestReadFileData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fsutils_test")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	content := []byte("0123456789")
	filename := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(filename, content, 0644)
	assert.NoError(t, err)

	t.Run("max=0", func(t *testing.T) {
		data, err := ReadFileData(filename, 0)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("max>0_smaller_than_file", func(t *testing.T) {
		data, err := ReadFileData(filename, 5)
		assert.NoError(t, err)
		assert.Equal(t, content[:5], data)
	})

	t.Run("max>0_larger_than_file", func(t *testing.T) {
		data, err := ReadFileData(filename, 20)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("max<0_absMax_smaller_than_file", func(t *testing.T) {
		// absMax = 3, file size = 10. Should read last 3 bytes.
		data, err := ReadFileData(filename, -3)
		assert.NoError(t, err)
		assert.Equal(t, content[7:], data)
	})

	t.Run("max<0_absMax_larger_than_file", func(t *testing.T) {
		// absMax = 20, file size = 10. Should read all bytes.
		data, err := ReadFileData(filename, -20)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("not_exists", func(t *testing.T) {
		_, err := ReadFileData(filepath.Join(tmpDir, "none.txt"), 0)
		assert.Error(t, err)

		_, err = ReadFileData(filepath.Join(tmpDir, "none.txt"), 10)
		assert.Error(t, err)

		_, err = ReadFileData(filepath.Join(tmpDir, "none.txt"), -10)
		assert.Error(t, err)
	})
}
