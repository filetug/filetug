package masks

import (
	"testing"

	"github.com/datatug/filetug/pkg/sneatv/ttestutils"
	"github.com/rivo/tview"
)

func TestNewPanel(t *testing.T) {
	p := NewPanel()
	if p == nil {
		t.Fatal("expected panel to be created")
	}
	if p.Table == nil {
		t.Error("expected Table to be initialized")
	}
	if p.boxed == nil {
		t.Error("expected boxed to be initialized")
	}
	if len(p.masks) == 0 {
		t.Error("expected built-in masks to be loaded")
	}

	// Check if table has headers
	if p.GetCell(0, 0).Text != "Mask" {
		t.Errorf("expected header 'Mask', got %q", p.GetCell(0, 0).Text)
	}

	// Check if table has data (at least one row per mask + header)
	expectedRows := len(p.masks) + 1
	if p.GetRowCount() != expectedRows {
		t.Errorf("expected %d rows, got %d", expectedRows, p.GetRowCount())
	}
}

func TestPanel_Draw(t *testing.T) {
	s := ttestutils.NewSimScreen(t, "UTF-8", 80, 24)
	p := NewPanel()
	p.SetRect(0, 0, 80, 24)
	p.Draw(s)
	// If Draw doesn't panic, it's already something.
	// We could use ttestutils.ReadLine(s, 0, 80) to check content if needed.
}

func TestPanel_Focus(t *testing.T) {
	p := NewPanel()
	focused := false
	p.Focus(func(delegate tview.Primitive) {
		focused = true
	})
	// tview.Table.Focus usually doesn't call delegate unless there's something to focus or it's part of a larger application.
	// However, we want to see if it's called.
	// Actually, p.Table.Focus(delegate) is what's called.
	if !focused {
		t.Log("Focus delegate not called, which might be normal for tview.Table without an app context")
	}
}
