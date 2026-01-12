package filetug

import (
	"testing"
)

func TestNewFavorites(t *testing.T) {
	f := newFavorites(nil)
	if f == nil {
		t.Fatal("f is nil")
	}
}
