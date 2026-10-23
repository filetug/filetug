package charmui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files/osfile"
)

// runProgram starts the Bubble Tea program. Tests replace it so both the
// success and failure exits of the MVP entry point can run without a terminal.
var runProgram = func(model tea.Model) error {
	program := tea.NewProgram(model)
	_, err := program.Run()
	return err
}

// RunLocal starts Filetug's direct Bubble Tea MVP entry point.
func RunLocal(path string) error {
	if path == "" {
		path = "."
	}
	store := osfile.NewStore(path)
	model, err := NewModel(path, store)
	if err != nil {
		return err
	}
	err = runProgram(model)
	if err != nil {
		return fmt.Errorf("run Bubble Tea UI: %w", err)
	}
	return nil
}
