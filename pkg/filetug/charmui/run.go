package charmui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/filetug/filetug/pkg/files/osfile"
)

// RunLocal starts Filetug's direct Bubble Tea MVP entry point.
func RunLocal(path string) error {
	store := osfile.NewStore(path)
	model, err := NewModel(path, store)
	if err != nil {
		return err
	}
	program := tea.NewProgram(model)
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("run Bubble Tea UI: %w", err)
	}
	return nil
}
