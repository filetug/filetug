package osfile

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore_GetDirReader(t *testing.T) {
	// Note: Cannot use t.Parallel() because subtest modifies global osOpen
	tempDir := t.TempDir()
	s := NewStore(tempDir)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		dr, err := s.GetDirReader(ctx, tempDir)
		assert.NoError(t, err)
		assert.NotNil(t, dr)
		defer func() {
			_ = dr.Close()
		}()

		entries, err := dr.Readdir()
		assert.NoError(t, err)
		assert.NotNil(t, entries)
	})

	t.Run("open_error", func(t *testing.T) {
		origOsOpen := osOpen
		defer func() { osOpen = origOsOpen }()
		osOpen = func(name string) (*os.File, error) {
			return nil, errors.New("open error")
		}

		dr, err := s.GetDirReader(ctx, tempDir)
		assert.Error(t, err)
		assert.Nil(t, dr)
	})
}

func TestDirReader_Close(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	s := NewStore(tempDir)
	dr, err := s.GetDirReader(context.Background(), tempDir)
	assert.NoError(t, err)

	err = dr.Close()
	assert.NoError(t, err)
}
