package filetug

import (
	"testing"

	"github.com/filetug/filetug/pkg/tviewmocks"
	"go.uber.org/mock/gomock"
)

func TestSetupApp(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	app := tviewmocks.NewMockApp(ctrl)
	expect := app.EXPECT()
	expect.QueueUpdateDraw(gomock.Any()).
		MinTimes(1).MaxTimes(2) // Should it be exactly 1?
	expect.EnableMouse(true)
	expect.SetRoot(gomock.Any(), true).Times(1)
	SetupApp(app)
}
