package filetug

import (
	"github.com/filetug/filetug/pkg/tviewmocks"
	"go.uber.org/mock/gomock"
)

func expectQueueUpdateDrawSyncMinMaxTimes(app *tviewmocks.MockApp, minTimes, maxTimes int) {
	app.EXPECT().QueueUpdateDraw(gomock.Any()).
		MinTimes(minTimes).
		MaxTimes(maxTimes).
		DoAndReturn(func(f func()) {
			f()
		})
}

func expectSetFocusMinMaxTimes(app *tviewmocks.MockApp, minTimes, maxTimes int) {
	app.EXPECT().SetFocus(gomock.Any()).
		MinTimes(minTimes).
		MaxTimes(maxTimes)
}
