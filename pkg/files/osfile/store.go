package osfile

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/filetug/filetug/pkg/files"
)

var osReadDir = os.ReadDir
var osHostname = os.Hostname
var osMkdir = os.Mkdir
var osCreate = os.Create
var osRemove = os.Remove

var _ files.Store = (*Store)(nil)

type Store struct {
	title string
	root  string
}

func (s Store) Delete(ctx context.Context, path string) error {
	_ = ctx
	return osRemove(path)
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

func (s Store) CreateDir(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return osMkdir(path, 0755)
}

func (s Store) CreateFile(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := osCreate(path)
	if err != nil {
		return err
	}
	return f.Close()
}

func NewStore(root string) *Store {
	if root == "" {
		_, _ = fmt.Fprintf(os.Stderr, "osfile store root is empty, defaulting to /\n")
		root = "/"
	}
	store := Store{root: root}
	var err error
	if store.title, err = osHostname(); err != nil {
		store.title = err.Error()
	}
	store.title = "🖥️" + store.title
	return &store
}
