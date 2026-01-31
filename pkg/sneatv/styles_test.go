package sneatv

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestDefaultBorderWithoutPadding_FocusBlur(t *testing.T) {
	t.Parallel()
	box := tview.NewBox()

	DefaultBorderWithoutPadding(box)
	assert.Equal(t, DefaultBlurBorderColor, box.GetBorderColor())

	box.Focus(func(p tview.Primitive) {})
	assert.Equal(t, DefaultFocusedBorderColor, box.GetBorderColor())

	box.Blur()
	assert.Equal(t, DefaultBlurBorderColor, box.GetBorderColor())
}
