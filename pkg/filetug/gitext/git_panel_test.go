package gitext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStatusPanel(t *testing.T) {
	panel := NewStatusPanel()

	assert.NotNil(t, panel)
	assert.NotNil(t, panel.Boxed)
	assert.NotNil(t, panel.flex)
	assert.Same(t, panel.flex, panel.BoxedContent)
	assert.Equal(t, "Git Status", panel.flex.GetTitle())
}

func TestStatusPanelSetDir(t *testing.T) {
	panel := NewStatusPanel()

	assert.NotPanics(t, func() {
		panel.SetDir("")
		panel.SetDir("some/dir")
	})
}
