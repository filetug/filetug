package tui

import (
	"testing"

	"github.com/tuigoff/tuigoff/pkg/theme"
)

func TestFileNameColor(t *testing.T) {
	defer theme.SetDark(theme.Dark)
	theme.SetDark(true)
	dark := theme.Hex(fileNameColor("main.GO"))
	if dark != "#00d7d7" && dark != "#00D7D7" {
		t.Fatalf("Go files are cyan on a dark theme: %s", dark)
	}
	theme.SetDark(false)
	if light := theme.Hex(fileNameColor("main.go")); light == dark {
		t.Fatal("the colour is darkened on a light theme")
	}
	if theme.Hex(fileNameColor("noextension")) != theme.Hex(theme.TextColor()) {
		t.Fatal("unknown extensions use the text colour")
	}
}
