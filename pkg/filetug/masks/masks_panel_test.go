package masks

import (
	"testing"

	"github.com/filetug/filetug/pkg/sneatv/ttestutils"
	"github.com/rivo/tview"
)

func TestNewPanel(t *testing.T) {
	t.Parallel()
	p := NewPanel()
	if p == nil {
		t.Fatal("expected panel to be created")
	}
	if p.table == nil {
		t.Error("expected Table to be initialized")
	}
	if p.Boxed == nil {
		t.Error("expected boxed to be initialized")
	}
	if len(p.masks) == 0 {
		t.Error("expected built-in masks to be loaded")
	}

	// Check if table has headers
	if p.table.GetCell(0, 0).Text != "Mask" {
		t.Errorf("expected header 'Mask', got %q", p.table.GetCell(0, 0).Text)
	}

	// Check if table has data (at least one row per mask + header)
	expectedRows := len(p.masks) + 1
	if p.table.GetRowCount() != expectedRows {
		t.Errorf("expected %d rows, got %d", expectedRows, p.table.GetRowCount())
	}
}

func TestPanel_Draw(t *testing.T) {
	t.Parallel()
	s := ttestutils.NewSimScreen(t, "UTF-8", 80, 24)
	p := NewPanel()
	p.SetRect(0, 0, 80, 24)
	p.Draw(s)
	// If Draw doesn't panic, it's already something.
	// We could use ttestutils.ReadLine(s, 0, 80) to check content if needed.
}

func TestPanel_Focus(t *testing.T) {
	t.Parallel()
	p := NewPanel()
	p.Focus(func(delegate tview.Primitive) {
	})
}
