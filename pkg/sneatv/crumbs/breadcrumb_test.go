package crumbs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBreadcrumb(t *testing.T) {
	t.Parallel()
	t.Run("with_action", func(t *testing.T) {
		actionCalled := false
		action := func() error {
			actionCalled = true
			return nil
		}
		bc := NewBreadcrumb("Title", action)
		assert.Equal(t, "Title", bc.GetTitle())
		err := bc.Action()
		assert.NoError(t, err)
		assert.True(t, actionCalled)
	})

	t.Run("without_action", func(t *testing.T) {
		bc := NewBreadcrumb("Title", nil)
		assert.Equal(t, "Title", bc.GetTitle())
		err := bc.Action()
		assert.NoError(t, err)
	})

	t.Run("with_error_action", func(t *testing.T) {
		expectedErr := errors.New("test error")
		action := func() error {
			return expectedErr
		}
		bc := NewBreadcrumb("Title", action)
		err := bc.Action()
		assert.Equal(t, expectedErr, err)
	})
}

func TestBreadcrumb_SetTitle(t *testing.T) {
	t.Parallel()
	bc := NewBreadcrumb("Old Title", nil)
	bc.SetTitle("New Title")
	assert.Equal(t, "New Title", bc.GetTitle())
}
