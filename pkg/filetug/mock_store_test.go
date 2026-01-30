package filetug

import (
	"net/url"
	"testing"

	"github.com/filetug/filetug/pkg/files"
	"go.uber.org/mock/gomock"
)

func newMockStore(t *testing.T) *files.MockStore {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return files.NewMockStore(ctrl)
}

func newMockStoreWithRoot(t *testing.T, rootURL url.URL) *files.MockStore {
	t.Helper()
	store := newMockStore(t)
	store.EXPECT().RootURL().Return(rootURL).AnyTimes()
	store.EXPECT().RootTitle().Return("Mock").AnyTimes()
	return store
}

func newMockStoreWithRootTitle(t *testing.T, rootURL url.URL, title string) *files.MockStore {
	t.Helper()
	store := newMockStore(t)
	store.EXPECT().RootURL().Return(rootURL).AnyTimes()
	store.EXPECT().RootTitle().Return(title).AnyTimes()
	return store
}
