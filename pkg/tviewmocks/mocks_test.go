package tviewmocks

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMockPrimitive(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	p := NewMockPrimitive(ctrl)
	assert.NotNil(t, p)

	p.EXPECT().HasFocus().Times(1).Return(true)
	assert.True(t, p.HasFocus())

	p.EXPECT().Focus(gomock.Any()).Times(1)
	p.Focus(func(primitive tview.Primitive) {
		assert.NotNil(t, primitive)
	})

	p.EXPECT().Blur().Times(1)
	p.Blur()

	p.EXPECT().Draw(gomock.Any()).Times(1)
	p.Draw(nil)

	p.EXPECT().InputHandler().Times(1)
	_ = p.InputHandler()

	p.EXPECT().GetRect().Times(1)
	_, _, _, _ = p.GetRect()

	p.EXPECT().MouseHandler().Times(1)
	_ = p.MouseHandler()

	p.EXPECT().PasteHandler().Times(1)
	_ = p.PasteHandler()

	p.EXPECT().SetRect(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	p.SetRect(0, 0, 0, 0)
}
