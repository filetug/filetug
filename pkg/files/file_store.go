package files

import (
	"net/url"
	"os"
)

type Store interface {
	RootTitle() string
	RootURL() url.URL
	ReadDir(name string) ([]os.DirEntry, error)
}
