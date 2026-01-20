package sneatest

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// helper to read a full line from the screen
func ReadLine(screen tcell.Screen, y, width int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		str, _, _ := screen.Get(x, y)
		if str == "" {
			// nothing drawn at this cell
			b.WriteRune(' ')
			continue
		}
		b.WriteString(str)
	}
	return b.String()
}

func NewSimScreen(t *testing.T, width, height int) tcell.Screen {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init simulation screen: %v", err)
	}
	s.SetSize(width, height)
	return s
}
