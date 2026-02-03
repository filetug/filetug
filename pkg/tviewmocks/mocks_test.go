package tviewmocks

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMockPrimitive(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := NewMockPrimitive(ctrl)
	assert.NotNil(t, p)
	p.EXPECT().Focus(gomock.Any())
	p.Focus(func(primitive tview.Primitive) {
		assert.NotNil(t, primitive)
	})
}
