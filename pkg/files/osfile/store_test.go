package osfile

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStore(t *testing.T) {
	origHostname := osHostname
	defer func() { osHostname = origHostname }()

	t.Run("valid_root", func(t *testing.T) {
		osHostname = func() (string, error) {
			return "test-host", nil
		}
		s := NewStore("/tmp")
		assert.NotNil(t, s)
		assert.Equal(t, "/tmp", s.root)
		assert.Equal(t, "🖥️test-host", s.title)
	})

	t.Run("hostname_error", func(t *testing.T) {
		osHostname = func() (string, error) {
			return "", errors.New("hostname error")
		}
		s := NewStore("/tmp")
		assert.NotNil(t, s)
		assert.Equal(t, "🖥️hostname error", s.title)
	})

	t.Run("empty_root_defaults", func(t *testing.T) {
		s := NewStore("")
		assert.NotNil(t, s)
		assert.Equal(t, "/", s.root)
	})
}

func TestStore_RootURL(t *testing.T) {
	s := NewStore("/tmp")
	u := s.RootURL()
	assert.Equal(t, "file", u.Scheme)
}

func TestStore_RootTitle(t *testing.T) {
	s := Store{title: "my-host.station"}
	assert.Equal(t, "my-host", s.RootTitle())

	s = Store{title: "my-host"}
	assert.Equal(t, "my-host", s.RootTitle())
}

func TestStore_ReadDir(t *testing.T) {
	origReadDir := osReadDir
	defer func() { osReadDir = origReadDir }()

	s := NewStore("/tmp")

	t.Run("success", func(t *testing.T) {
		osReadDir = func(name string) ([]os.DirEntry, error) {
			return []os.DirEntry{}, nil
		}
		entries, err := s.ReadDir(context.Background(), "/tmp")
		assert.NoError(t, err)
		assert.NotNil(t, entries)
	})

	t.Run("context_cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		entries, err := s.ReadDir(ctx, "/tmp")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Nil(t, entries)
	})

	t.Run("read_error", func(t *testing.T) {
		osReadDir = func(name string) ([]os.DirEntry, error) {
			return nil, errors.New("read error")
		}
		entries, err := s.ReadDir(context.Background(), "/tmp")
		assert.Error(t, err)
		assert.Nil(t, entries)
	})
}

func TestStore_CreateDir_CreateFile_Delete(t *testing.T) {
	origMkdir := osMkdir
	origCreate := osCreate
	origRemove := osRemove
	defer func() {
		osMkdir = origMkdir
		osCreate = origCreate
		osRemove = origRemove
	}()

	tempDir, err := os.MkdirTemp("", "osfile_test")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	s := NewStore(tempDir)
	ctx := context.Background()

	t.Run("CreateDir success", func(t *testing.T) {
		dirPath := tempDir + "/newdir"
		err := s.CreateDir(ctx, dirPath)
		assert.NoError(t, err)
		info, err := os.Stat(dirPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("CreateDir error", func(t *testing.T) {
		osMkdir = func(path string, perm os.FileMode) error {
			return errors.New("mkdir error")
		}
		err := s.CreateDir(ctx, tempDir+"/error")
		assert.Error(t, err)
		osMkdir = origMkdir
	})

	t.Run("CreateDir context cancelled", func(t *testing.T) {
		ctxC, cancel := context.WithCancel(ctx)
		cancel()
		err := s.CreateDir(ctxC, tempDir+"/cancelled")
		assert.Error(t, err)
	})

	t.Run("CreateFile success", func(t *testing.T) {
		filePath := tempDir + "/newfile.txt"
		err := s.CreateFile(ctx, filePath)
		assert.NoError(t, err)
		_, err = os.Stat(filePath)
		assert.NoError(t, err)
	})

	t.Run("CreateFile context cancelled", func(t *testing.T) {
		ctxC, cancel := context.WithCancel(ctx)
		cancel()
		err := s.CreateFile(ctxC, tempDir+"/cancelled.txt")
		assert.Error(t, err)
	})

	t.Run("CreateFile error", func(t *testing.T) {
		err := s.CreateFile(ctx, tempDir+"/nonexistent/file.txt")
		assert.Error(t, err)
	})

	t.Run("Delete success", func(t *testing.T) {
		filePath := tempDir + "/todelete.txt"
		err := os.WriteFile(filePath, []byte("test"), 0644)
		assert.NoError(t, err)

		err = s.Delete(ctx, filePath)
		assert.NoError(t, err)

		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Delete error", func(t *testing.T) {
		osRemove = func(name string) error {
			return errors.New("remove error")
		}
		err := s.Delete(ctx, "any")
		assert.Error(t, err)
		osRemove = origRemove
	})
}
