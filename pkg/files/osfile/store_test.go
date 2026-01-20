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

	t.Run("empty_root_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			NewStore("")
		})
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
