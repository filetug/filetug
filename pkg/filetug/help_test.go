package filetug

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestShowHelpModal(t *testing.T) {
	realApp := tview.NewApplication()
	nav := NewNavigator(realApp)
	showHelpModal(nav)
	assert.NotNil(t, nav)
}
