package files

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
)

// noinspection GoUnusedGlobalVariable // used by other packages
var ErrNotImplemented = errors.New("not implemented")

// noinspection GoUnusedGlobalVariable // used by other packages
var ErrNotSupported = errors.New("not supported")

type Store interface {
	RootTitle() string
	RootURL() url.URL
	GetDirReader(ctx context.Context, path string) (DirReader, error)
	ReadDir(ctx context.Context, path string) ([]os.DirEntry, error)
	Delete(ctx context.Context, path string) error // TODO(unsure): should it be Remove to match os.Remove?
	CreateDir(ctx context.Context, path string) error
	CreateFile(ctx context.Context, path string) error
}

type DirReader interface {
	io.Closer
	Readdir() ([]os.FileInfo, error)
}
