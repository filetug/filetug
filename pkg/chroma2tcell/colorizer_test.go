package chroma2tcell

import (
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func TestColorizeYAMLForTview(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s, err := ColorizeYAMLForTview("", lexers.Get)
		assert.NoError(t, err)
		assert.Equal(t, "", s)
	})

	t.Run("simple_yaml", func(t *testing.T) {
		s, err := ColorizeYAMLForTview("key: value", lexers.Get)
		assert.NoError(t, err)
		assert.Contains(t, s, "[")
		assert.Contains(t, s, "key")
		assert.Contains(t, s, "value")
	})

	t.Run("lexer_not_found", func(t *testing.T) {
		s, err := ColorizeYAMLForTview("key: value", func(string) chroma.Lexer { return nil })
		assert.NoError(t, err)
		assert.Contains(t, s, "key: value")
	})
}

func TestColorize(t *testing.T) {
	// Note: Cannot use t.Parallel() because subtests modify global getStyle and getFallbackStyle
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

	t.Run("getFallbackStyle", func(t *testing.T) {
		actual := getFallbackStyle()
		assert.Equal(t, styles.Fallback, actual)
	})

	t.Run("unknown_style", func(t *testing.T) {
		lexer := lexers.Get("go")
		getStyleCalls := 0
		fallbackCalls := 0
		oldGetFallbackStyle := getFallbackStyle
		defer func() {
			getFallbackStyle = oldGetFallbackStyle
		}()
		getStyle = func(name string) *chroma.Style {
			getStyleCalls++
			return nil
		}
		getFallbackStyle = func() *chroma.Style {
			fallbackCalls++
			return styles.Fallback
		}
		s, err := Colorize("", "unknown_style", lexer)
		assert.NoError(t, err)
		assert.Equal(t, 1, getStyleCalls)
		assert.Equal(t, 1, fallbackCalls)
		assert.Equal(t, "", s)
	})

	t.Run("token_with_no_color", func(t *testing.T) {
		lexer := lexers.Get("go")
		// The style "swapoff" might not have colors for everything
		s, err := Colorize("package main", "swapoff", lexer)
		assert.NoError(t, err)
		assert.Contains(t, s, "package")
	})

	t.Run("tokenise_error", func(t *testing.T) {
		lexer := &mockLexer{err: fmt.Errorf("tokenise error")}
		_, err := Colorize("text", "dracula", lexer)
		assert.Error(t, err)
	})

	t.Run("zero_color", func(t *testing.T) {
		lexer := &mockLexer{
			tokens: []chroma.Token{
				{Type: chroma.TokenType(-1), Value: "plain text"},
			},
		}

		const zeroStyleName = "zero"
		zeroStyle := &chroma.Style{
			Name: "zero",
		}

		oldGetStyle := getStyle
		defer func() {
			getStyle = oldGetStyle
		}()

		getStyle = func(name string) *chroma.Style {
			return zeroStyle
		}

		const input = "plain text"
		s, err := Colorize(input, zeroStyleName, lexer)
		assert.NoError(t, err)
		assert.Equal(t, input, s)
	})
}

type mockLexer struct {
	tokens []chroma.Token
	err    error
}

func (m *mockLexer) Tokenise(options *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	_, _ = options, text
	if m.err != nil {
		return nil, m.err
	}
	return chroma.Literator(m.tokens...), nil
}

func (m *mockLexer) Config() *chroma.Config {
	return nil
}

func (m *mockLexer) SetRegistry(_ *chroma.LexerRegistry) chroma.Lexer {
	return m
}

func (m *mockLexer) SetAnalyser(analyser func(text string) float32) chroma.Lexer {
	_ = analyser
	return m
}

func (m *mockLexer) AnalyseText(_ string) float32 {
	return 0
}
