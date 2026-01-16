package osfile

import (
	"net/url"
	"os"
	"strings"

	"github.com/datatug/filetug/pkg/files"
)

var osReadDir = os.ReadDir

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

func (s Store) ReadDir(name string) ([]os.DirEntry, error) {
	return osReadDir(name)
}

func NewStore(root string) *Store {
	if root == "" {
		panic("root is empty")
	}
	store := Store{root: root}
	store.title, _ = os.Hostname()
	return &store
}
