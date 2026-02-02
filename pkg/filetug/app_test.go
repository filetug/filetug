package filetug

import (
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"go.uber.org/mock/gomock"
)

func expectQueueUpdateDrawSyncTimes(app *navigator.MockApp, times int) {
	app.EXPECT().QueueUpdateDraw(gomock.Any()).Times(times).DoAndReturn(func(f func()) {
		f()
	})
}

func expectQueueUpdateDrawSyncMinMaxTimes(app *navigator.MockApp, minTimes, maxTimes int) {
	app.EXPECT().QueueUpdateDraw(gomock.Any()).
		MinTimes(minTimes).
		MaxTimes(maxTimes).
		DoAndReturn(func(f func()) {
			f()
		})
}
