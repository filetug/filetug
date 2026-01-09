package chroma2tcell

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestColorizeYAMLForTview(t *testing.T) {
	s, err := ColorizeYAMLForTview("")
	assert.NoError(t, err)
	assert.Equal(t, "", s)
}
