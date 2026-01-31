package filetug

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestGetColorByFileName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fileName string
		want     tcell.Color
	}{
		{"exe", "test.exe", tcell.ColorRed},
		{"go", "main.go", tcell.ColorAqua},
		{"cpp", "main.cpp", tcell.ColorDodgerBlue},
		{"sql", "query.sql", tcell.ColorSpringGreen},
		{"html", "index.html", tcell.ColorOrangeRed},
		{"json", "data.json", tcell.ColorGold},
		{"jpg", "image.jpg", tcell.ColorMediumPurple},
		{"png", "photo.png", tcell.ColorMediumPurple},
		{"mov", "video.mov", tcell.ColorLightSalmon},
		{"log", "app.log", tcell.ColorRosyBrown},
		{"xls", "sheet.xls", tcell.ColorGreen},
		{"doc", "word.doc", tcell.ColorBlue},
		{"no_ext", "README", tcell.ColorWhiteSmoke},
		{"unknown_ext", "config.unknown", tcell.ColorWhiteSmoke},
		{"hidden_go", ".go", tcell.ColorAqua},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetColorByFileExt(tt.fileName); got != tt.want {
				t.Errorf("GetColorByFileExt() = %v, want %v", got, tt.want)
			}
		})
	}
}
