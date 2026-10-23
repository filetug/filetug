package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func TestPreviewPane(t *testing.T) {
	p := newPreviewPane()
	p.SetSize(30, 8)
	p.Loading("a.go")
	if p.Title() != "a.go" || !strings.Contains(p.View(), "Loading…") {
		t.Fatal("loading")
	}
	p.Show(previewMsg{Title: "a.go", Size: "3B", Modified: "today", Body: "hello"})
	view := p.View()
	for _, want := range []string{"Size", "3B", "Modified", "today", "hello", "──"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in %q", want, view)
		}
	}
	if p.Text() != "hello" || p.loading {
		t.Fatal("shown")
	}
	p.Show(previewMsg{Title: "b", Err: errors.New("boom")})
	if p.Text() != "boom" {
		t.Fatal("errors are shown as text")
	}
}

func TestPreviewPaneScrolls(t *testing.T) {
	p := newPreviewPane()
	p.SetSize(20, 6)
	p.Show(previewMsg{Body: strings.Repeat("line\n", 30)})
	if !p.AtEdge(widgets.Up) || p.AtEdge(widgets.Down) {
		t.Fatal("at the top")
	}
	p.Focus()
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if p.AtEdge(widgets.Up) {
		t.Fatal("scrolled")
	}
	p.Blur()
	p2, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if p2.AtEdge(widgets.Up) {
		t.Fatal("a blurred pane does not scroll")
	}
}
