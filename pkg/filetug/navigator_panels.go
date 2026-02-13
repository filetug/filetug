package filetug

import (
	"github.com/filetug/filetug/pkg/filetug/masks"
)

func (nav *Navigator) showMasks() {
	if nav.masks == nil {
		nav.masks = masks.NewPanel()
	}
	if nav.right != nil {
		nav.right.SetContent(nav.masks)
	}
}

func (nav *Navigator) showNewPanel() {
	if nav.newPanel != nil {
		nav.newPanel.Show()
	}
}
