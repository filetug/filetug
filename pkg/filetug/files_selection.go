package filetug

import (
	"fmt"
	"os"

	"github.com/filetug/filetug/pkg/files"
)

// GetCurrentEntry returns the currently selected entry in the files panel.
// It handles the parent directory entry and regular file/directory entries.
func (f *filesPanel) GetCurrentEntry() files.EntryWithDirPath {
	row, _ := f.table.GetSelection()
	i := row - 1
	if i < 0 || i >= len(f.rows.VisibleEntries) {
		return nil
	}
	entry := f.rows.VisibleEntries[i]
	if entry.DirPath() == "" {
		if f.rows.Dir == nil {
			_, _ = fmt.Fprintf(os.Stderr, "files panel missing dir path for entry %q\n", entry.Name())
			return nil
		}
		entry = files.NewEntryWithDirPath(entry, f.rows.Dir.Path())
	}

	return entry
}

// SetCurrentFile sets the current filename and selects it in the table if found.
func (f *filesPanel) SetCurrentFile(name string) {
	f.currentFileName = name
	f.selectCurrentFile()
}

// selectCurrentFile finds and selects the row containing the current filename.
func (f *filesPanel) selectCurrentFile() {
	if f.currentFileName == "" || f.rows == nil {
		return
	}
	for i, entry := range f.rows.AllEntries {
		if entry.Name() == f.currentFileName {
			row, _ := f.table.GetSelection()
			if row != i+1 {
				f.table.Select(i+1, 0)
			}
			return
		}
	}
}

// selectionChangedNavFunc handles selection changes for navigation.
func (f *filesPanel) selectionChangedNavFunc(row, _ int) {
	entry := f.entryFromRow(row)
	if entry == nil {
		return
	}
	f.updatePreviewForEntry(entry)
}

// selectionChanged handles selection changes in the table.
func (f *filesPanel) selectionChanged(row, _ int) {
	entry := f.entryFromRow(row)
	if entry == nil {
		if f.nav != nil && f.nav.previewer != nil {
			f.nav.previewer.SetText("cell has no reference")
		}
		return
	}
	f.updatePreviewForEntry(entry)
}

// entryFromRow retrieves the entry corresponding to the given row in the table.
// Row 0 is special (parent directory), rows 1+ correspond to visible entries.
func (f *filesPanel) entryFromRow(row int) files.EntryWithDirPath {
	if f.table == nil || f.rows == nil {
		return nil
	}
	if row == 0 {
		cell := f.table.GetCell(row, 0)
		if cell != nil {
			ref := cell.GetReference()
			if ref != nil {
				entry, ok := ref.(files.EntryWithDirPath)
				if ok {
					return entry
				}
			}
		}
		// Fallback to generating reference if not set
		return f.rows.getTopRowEntry()
	}
	i := row - 1
	if i < 0 || i >= len(f.rows.VisibleEntries) {
		return nil
	}
	entry := f.rows.VisibleEntries[i]
	if entry.DirPath() == "" {
		if f.rows.Dir != nil {
			entry = files.NewEntryWithDirPath(entry, f.rows.Dir.Path())
		}
	}
	return entry
}
