package filetug

import (
	"testing"

	"github.com/filetug/filetug/pkg/tviewmocks"
	"go.uber.org/mock/gomock"
)

func newNavigatorForTest(t *testing.T, options ...NavigatorOption) (nav *Navigator, app *tviewmocks.MockApp, ctrl *gomock.Controller) {
	ctrl = gomock.NewController(t)
	app = tviewmocks.NewMockApp(ctrl)
	app.EXPECT().QueueUpdateDraw(gomock.Any()).AnyTimes().DoAndReturn(func(f func()) {
		if f != nil {
			f()
		}
	})
	app.EXPECT().SetFocus(gomock.Any()).AnyTimes()
	app.EXPECT().SetRoot(gomock.Any(), gomock.Any()).AnyTimes()
	return NewNavigator(app, options...), app, ctrl
}
