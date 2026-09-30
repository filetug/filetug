package tui

import (
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/tuigoff/tuigoff/pkg/theme"
)

// fileColors maps a lower-case file extension (without the dot) to the hex
// colour of its name in the file list.
var fileColors = map[string]string{
	"exe": "#FF5555", "go": "#00D7D7", "cpp": "#1E90FF", "c": "#1E90FF", "h": "#1E90FF",
	"cs": "#5FD75F", "js": "#E9C34B", "ts": "#00BFFF", "html": "#FF6A33", "css": "#EE82EE",
	"sql": "#00FF7F", "json": "#FFD700", "xml": "#F0E68C", "yaml": "#F0E68C", "yml": "#F0E68C",
	"md": "#FFE4C4", "py": "#90EE90", "rb": "#FF5555", "php": "#B48EAD", "rs": "#FFA500",
	"sh": "#5FD75F", "bat": "#CD5C5C", "txt": "#E0E0E0", "csv": "#90EE90",
	"jpg": "#9370DB", "jpeg": "#9370DB", "png": "#9370DB", "gif": "#9370DB", "webp": "#9370DB",
	"mov": "#FFA07A", "mp4": "#FFA07A", "log": "#BC8F8F",
	"xls": "#5FD75F", "xlsx": "#5FD75F", "doc": "#6C8CFF", "docx": "#6C8CFF",
}

// fileNameColor returns the colour of a file name, chosen by its extension.
// On a light theme the colour is darkened so it stays readable. Unknown
// extensions use the theme's text colour.
func fileNameColor(name string) color.Color {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	hex, ok := fileColors[ext]
	if !ok {
		return theme.TextColor()
	}
	c := lipgloss.Color(hex)
	if theme.Dark {
		return c
	}
	return lipgloss.Darken(c, 0.45)
}
