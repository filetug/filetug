package filetug

import "testing"

func Test_createLeft(t *testing.T) {
	nav := &Navigator{}
	nav.favorites = newFavorites(nav)
	createLeft(nav)
}
