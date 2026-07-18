package filetug

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestInputCapture_OptionRuneWithoutModAlt covers the `r != event.Rune()`
// operand of inputCapture's alt-branch guard: a macOS Option-key rune ('å')
// arrives WITHOUT the ModAlt modifier, so the left operand is false and the
// normalized rune (differing from the original) is what opens the branch. 'a'
// then hits the inner default, returning the event unchanged.
func TestInputCapture_OptionRuneWithoutModAlt(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	event := tcell.NewEventKey(tcell.KeyRune, 'å', tcell.ModNone)
	if got := nav.inputCapture(event); got != event {
		t.Errorf("inputCapture(Option-a, no ModAlt) = %v, want the event returned unchanged", got)
	}
}

func TestMacOSOptionRune(t *testing.T) {
	t.Parallel()
	// Every Option+key mapping the function recognises.
	cases := []struct {
		in   rune
		want rune
	}{
		{'å', 'a'},
		{'∫', 'b'},
		{'ç', 'c'},
		{'∂', 'd'},
		{'ƒ', 'f'},
		{'©', 'g'},
		{'˙', 'h'},
		{'∆', 'j'},
		{'˚', 'k'},
		{'¬', 'l'},
		{'µ', 'm'},
		{'ø', 'o'},
		{'π', 'p'},
		{'œ', 'q'},
		{'®', 'r'},
		{'ß', 's'},
		{'†', 't'},
		{'√', 'v'},
		{'∑', 'w'},
		{'≈', 'x'},
		{'¥', 'y'},
		{'Ω', 'z'},
		{'º', '0'},
		{'–', '-'},
		{'≠', '='},
		{'÷', '/'},
	}
	for _, tc := range cases {
		got, ok := macOSOptionRune(tc.in)
		if !ok {
			t.Errorf("macOSOptionRune(%q): ok=false, want true", tc.in)
			continue
		}
		if got != tc.want {
			t.Errorf("macOSOptionRune(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// A rune that is not an Option-mapped char is returned unchanged, ok=false.
	if got, ok := macOSOptionRune('a'); ok || got != 'a' {
		t.Errorf("macOSOptionRune('a') = (%q, %v), want ('a', false)", got, ok)
	}
}
