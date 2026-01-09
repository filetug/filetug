package chroma2tcell

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func Colorize(text, styleName string, lexer chroma.Lexer) (string, error) {
	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return "", err
	}

	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
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
		sb.WriteString("[" + color.Colour.String() + "]")
		sb.WriteString(token.Value)
		sb.WriteString("[-]")
	}

	return sb.String(), nil
}

func ColorizeYAMLForTview(yamlStr string) (string, error) {
	lexer := lexers.Get("yaml")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return Colorize(yamlStr, "dracula", lexer)
}
