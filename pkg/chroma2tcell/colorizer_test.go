package chroma2tcell

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

func TestColorizeYAMLForTview(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := ColorizeYAMLForTview("")
		assert.NoError(t, err)
		assert.Equal(t, "", s)
	})

	t.Run("simple_yaml", func(t *testing.T) {
		s, err := ColorizeYAMLForTview("key: value")
		assert.NoError(t, err)
		assert.Contains(t, s, "[")
		assert.Contains(t, s, "key")
		assert.Contains(t, s, "value")
	})
}

func TestColorize(t *testing.T) {
	t.Run("invalid_lexer", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		_, _ = Colorize("text", "dracula", nil)
	})

	t.Run("with_lexer", func(t *testing.T) {
		lexer := lexers.Get("go")
		s, err := Colorize("package main", "dracula", lexer)
		assert.NoError(t, err)
		assert.Contains(t, s, "package")
	})

	t.Run("unknown_style", func(t *testing.T) {
		lexer := lexers.Get("go")
		s, err := Colorize("package main", "unknown-style", lexer)
		assert.NoError(t, err)
		assert.Contains(t, s, "package")
	})
}
