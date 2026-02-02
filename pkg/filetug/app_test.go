package filetug

import (
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"go.uber.org/mock/gomock"
)

func expectQueueUpdateDrawSync(app *navigator.MockApp, times int) {
	app.EXPECT().QueueUpdateDraw(gomock.Any()).Times(times).DoAndReturn(func(f func()) {
		f()
	})
}
