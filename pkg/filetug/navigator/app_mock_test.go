package navigator

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAppProxy_WithMockApp(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	a := NewMockApp(ctrl)
	assert.NotNil(t, a)

	root := tview.NewTextView()
	//expect := a.EXPECT()

	updateCalled := false

	updater := func() {
		updateCalled = true
	}

	a.EXPECT().EnableMouse(true)
	a.EXPECT().Run().Times(1).Return(nil)
	a.EXPECT().SetRoot(gomock.Any(), gomock.Any()).Times(1)
	a.EXPECT().SetFocus(root).Times(1)
	a.EXPECT().Stop().Times(1)
	a.EXPECT().QueueUpdateDraw(gomock.Any()).Times(1).Do(func(f func()) { f() })

	a.EnableMouse(true)
	err := a.Run()
	assert.Nil(t, err)
	a.SetRoot(root, true)
	a.SetFocus(root)
	a.QueueUpdateDraw(updater)
	assert.True(t, updateCalled)
	a.Stop()
}
