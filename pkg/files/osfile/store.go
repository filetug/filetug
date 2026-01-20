package osfile

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/datatug/filetug/pkg/files"
)

var osReadDir = os.ReadDir
var osHostname = os.Hostname

var _ files.Store = (*Store)(nil)

type Store struct {
	title string
	root  string
}

func (s Store) RootURL() url.URL {
	return url.URL{
		Scheme: "file",
	}
}

func (s Store) RootTitle() string {
	return strings.TrimSuffix(s.title, ".station")
}

func (s Store) ReadDir(ctx context.Context, name string) ([]os.DirEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return osReadDir(name)
}

func NewStore(root string) *Store {
	if root == "" {
		panic("root is empty")
	}
	store := Store{root: root}
	var err error
	if store.title, err = osHostname(); err != nil {
		store.title = err.Error()
	}
	store.title = "🖥️" + store.title
	return &store
}
