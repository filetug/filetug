package osfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalDir(t *testing.T) {
	t.Parallel()
	fullPath := "/tmp"
	dir := NewLocalDir(fullPath)

	assert.NotNil(t, dir)
	assert.Equal(t, fullPath, dir.Path())

	store, ok := dir.Store().(*Store)
	assert.True(t, ok)
	assert.Same(t, localFileStore, store)
	assert.Equal(t, "/", store.root)

	children := dir.Children()
	assert.Nil(t, children)
}
