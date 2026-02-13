package viewers

import (
	"github.com/filetug/filetug/pkg/filetug/ftui"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/gdamore/tcell/v2"
)

// DirSummaryOption is a functional option for configuring DirPreviewer.
type DirSummaryOption func(*DirPreviewer)

// WithDirSummaryFilterSetter sets the filter setter function.
func WithDirSummaryFilterSetter(setter func(ftui.Filter)) DirSummaryOption {
	return func(d *DirPreviewer) {
		d.setFilter = setter
	}
}

// WithDirSummaryFocusLeft sets the focus left function.
func WithDirSummaryFocusLeft(setter func()) DirSummaryOption {
	return func(d *DirPreviewer) {
		d.focusLeft = setter
	}
}

// WithDirSummaryQueueUpdateDraw sets the queue update draw function.
func WithDirSummaryQueueUpdateDraw(setter navigator.UpdateDrawQueuer) DirSummaryOption {
	return func(d *DirPreviewer) {
		d.queueUpdateDraw = setter
	}
}

// WithDirSummaryColorByExt sets the color by extension function.
func WithDirSummaryColorByExt(setter func(string) tcell.Color) DirSummaryOption {
	return func(d *DirPreviewer) {
		d.colorByExt = setter
	}
}
