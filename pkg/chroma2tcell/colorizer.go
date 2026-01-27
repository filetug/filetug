package chroma2tcell

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var getStyle = styles.Get

var getFallbackStyle = func() *chroma.Style {
	return styles.Fallback
}

func Colorize(text, styleName string, lexer chroma.Lexer) (string, error) {
	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return "", err
	}

	style := getStyle(styleName)
	if style == nil {
		style = getFallbackStyle()
	}

	var sb strings.Builder
	for _, token := range iterator.Tokens() {
		color := style.Get(token.Type)
		if color.IsZero() {
			sb.WriteString(token.Value)
			continue
		}

		// Map Chroma color to tview [color] tag
		// simple approximation: use hex
		colorText := color.Colour.String()
		sb.WriteString("[" + colorText + "]")
		sb.WriteString(token.Value)
		sb.WriteString("[-]")
	}

	return sb.String(), nil
}

func ColorizeYAMLForTview(yamlStr string, getLexer func(string) chroma.Lexer) (string, error) {
	lexer := getLexer("yaml")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return Colorize(yamlStr, "dracula", lexer)
}
