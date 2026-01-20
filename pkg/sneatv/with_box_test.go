package sneatv

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestWithBoxType(t *testing.T) {
	inner := tview.NewBox()
	box := tview.NewBox()
	wb := WithBoxType[*tview.Box]{
		Primitive: inner,
		Box:       box,
	}

	assert.Equal(t, box, wb.GetBox())
	assert.Equal(t, inner, wb.GetPrimitive())
}

func TestWithDefaultBorders(t *testing.T) {
	inner := tview.NewBox()
	box := tview.NewBox()
	wb := WithDefaultBorders(inner, box)

	assert.Equal(t, inner, wb.GetPrimitive())
	assert.Equal(t, box, wb.GetBox())
}

func TestWithBordersWithoutPadding(t *testing.T) {
	inner := tview.NewBox()
	box := tview.NewBox()
	wb := WithBordersWithoutPadding(inner, box)

	assert.Equal(t, inner, wb.GetPrimitive())
	assert.Equal(t, box, wb.GetBox())
}

func TestWithBoxWithoutBorder(t *testing.T) {
	inner := tview.NewBox()
	box := tview.NewBox()
	wb := WithBoxWithoutBorder(inner, box)

	assert.Equal(t, inner, wb.GetPrimitive())
	assert.Equal(t, box, wb.GetBox())
}

func TestSetPanelTitle(t *testing.T) {
	box := tview.NewBox()
	SetPanelTitle(box, "Test Title")
	assert.Equal(t, "Test Title", box.GetTitle())
}
