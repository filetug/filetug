package sneatv

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestPaddedBox(t *testing.T) {
	t.Parallel()
	content := tview.NewBox()
	title := "Test Title"
	paddingTop, paddingBottom, paddingLeft, paddingRight := 1, 2, 3, 4

	wb := PaddedBox(content, title, paddingTop, paddingBottom, paddingLeft, paddingRight)

	assert.NotNil(t, wb)
	assert.NotNil(t, wb.Box)
	assert.Equal(t, title, wb.Box.GetTitle())

	flex := wb.GetPrimitive()
	assert.NotNil(t, flex)
	// flex has 2 items: box and padded flex
	assert.Equal(t, 2, flex.GetItemCount())
}
