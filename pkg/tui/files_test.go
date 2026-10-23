package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files"
	"github.com/tuigoff/tuigoff/pkg/grid"
	"github.com/tuigoff/tuigoff/pkg/widgets"
)

func TestFilesPaneStates(t *testing.T) {
	p := newFilesPane()
	p.SetSize(40, 6)
	p.SetError(errors.New("denied"))
	if got := p.View(true); !strings.Contains(got, "denied") {
		t.Fatalf("an error: %q", got)
	}
	dir := files.NewDirContext(nil, "/", nil)
	p.SetDir(dir, false, "")
	if got := p.View(true); !strings.Contains(got, "No entries") {
		t.Fatalf("an empty directory: %q", got)
	}
	if p.Current() != nil {
		t.Fatal("no entry to be current")
	}
	if _, ok := p.CurrentRef(); ok {
		t.Fatal("no reference either")
	}

	dir = files.NewDirContext(nil, "/d", []os.DirEntry{files.NewDirEntry("a", false), files.NewDirEntry("b", false)})
	p.SetDir(dir, false, "b")
	if p.Current().Name() != "b" {
		t.Fatalf("the named entry is selected: %v", p.Current())
	}
	if ref, ok := p.CurrentRef(); !ok || ref.Entry.Name() != "b" {
		t.Fatal("CurrentRef")
	}
	p.SetDir(dir, false, "missing")
	if ref, _ := p.CurrentRef(); !ref.Parent {
		t.Fatal("an unknown name selects the first row")
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !p.AtEdge(widgets.Left) || p.AtEdge(widgets.Down) {
		t.Fatal("AtEdge follows the grid")
	}
}

func TestFilesFooter(t *testing.T) {
	st := &filesState{marks: map[string]bool{"x": true}}
	st.rows = newFileRows(files.NewDirContext(nil, "/d", []os.DirEntry{files.NewDirEntry("a", false)}), false, st.marks)
	if got := st.footer(nil, ""); got != "1 item · 1 selected" {
		t.Fatalf("%q", got)
	}
	st.rows = newFileRows(files.NewDirContext(nil, "/d", nil), false, st.marks)
	st.marks = map[string]bool{}
	if got := st.footer(nil, ""); got != "0 items" {
		t.Fatalf("%q", got)
	}
}

func TestCellStyle(t *testing.T) {
	entry := files.NewEntryWithDirPath(files.NewDirEntry("a.go", false), "/d")
	cases := []struct {
		row    grid.Row
		column int
	}{
		{grid.Row{}, nameColumn},
		{grid.Row{Ref: rowRef{Entry: entry, Err: errors.New("x")}}, modifiedColumn},
		{grid.Row{Ref: rowRef{Entry: entry, IsDir: true}}, nameColumn},
		{grid.Row{Ref: rowRef{Entry: entry}}, nameColumn},
		{grid.Row{Ref: rowRef{Entry: entry}}, sizeColumn},
	}
	for i, c := range cases {
		_ = cellStyle(c.row, c.column, nil)
		_ = i
	}
}

func TestToggleMarkIgnoresForeignRows(t *testing.T) {
	st := &filesState{marks: map[string]bool{}}
	st.toggleMark(grid.Row{})
	st.toggleMark(grid.Row{Ref: rowRef{Parent: true}})
	if len(st.marks) != 0 {
		t.Fatal("only entries can be marked")
	}
}

func TestClaimKey(t *testing.T) {
	st := &filesState{marks: map[string]bool{}}
	p := newFilesPane()
	if cmd, handled := st.claimKey(p.grid, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}); cmd != nil || !handled {
		t.Fatal("Space is claimed even when there is no row")
	}
	if cmd, handled := st.claimKey(p.grid, tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil || handled {
		t.Fatal("other keys are the grid's")
	}
	if cmd, handled := st.claimKey(p.grid, tea.KeyPressMsg{Code: '/', Text: "/"}); cmd == nil || !handled {
		t.Fatal("/ goes to the root")
	}
	if cmd, handled := st.claimKey(p.grid, tea.KeyPressMsg{Code: '`', Text: "`"}); cmd == nil || !handled {
		t.Fatal("` goes home")
	}
}
